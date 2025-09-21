package main

import (
	"context"
	"flag"
	"io/fs"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	prom "github.com/hertz-contrib/monitor-prometheus"

	"github.com/gogogo1024/assist-fusion/internal/common"
	"github.com/gogogo1024/assist-fusion/internal/gateway"
	rpcClients "github.com/gogogo1024/assist-fusion/internal/gateway/rpc"
	"github.com/gogogo1024/assist-fusion/internal/kb"
	esrepo "github.com/gogogo1024/assist-fusion/internal/kb/esrepo"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	"github.com/gogogo1024/assist-fusion/kitex_gen/ai/aiservice"
	"github.com/gogogo1024/assist-fusion/kitex_gen/kb/kbservice"
	gwerrors "github.com/gogogo1024/assist-fusion/services/gateway/internal/errors"
	router "github.com/gogogo1024/assist-fusion/services/gateway/internal/router"
)

// error message constants migrated to gwerrors package
const (
	notFoundMsg      = gwerrors.MsgNotFound
	badRequestMsg    = gwerrors.MsgBadRequest
	kbUnavailableMsg = gwerrors.MsgKBUnavailable
	internalErrMsg   = gwerrors.MsgInternal
)

// common header / path literal constants (lint de-dup)
const (
	headerContentType    = "Content-Type"
	contentTypeTextPlain = "text/plain; charset=utf-8"
)

// ticket paths moved into router/paths.go

var esInitOK bool
var esRepoInstance interface{ Ping(context.Context) error } // optional stored when using ES
var promOnce sync.Once

// removed unused prometheusTracerEnabled (previously used to indicate metrics server setup)

func main() {
	cfg := common.LoadConfig()
	h := BuildServer(cfg)
	log.Printf("gateway listening on %s", getAddr(cfg))
	h.Spin()
}

func getAddr(cfg *common.Config) string {
	if cfg.HTTPAddr != "" {
		return cfg.HTTPAddr
	}
	if v := os.Getenv("TICKET_ADDR"); v != "" { // backward compat env name
		return v
	}
	return ":8081"
}

// isGoTest returns true when running under `go test`.
func isGoTest() bool {
	return flag.Lookup("test.v") != nil
}

// BuildServer assembles the Hertz server with all routes for reuse in tests.
func BuildServer(cfg *common.Config) *server.Hertz {
	common.InitLogger()
	common.InitHertzLogger()
	repo := common.NewMemoryTicketRepo()
	var kbRepo kb.Repo
	if cfg.KBBackend == "es" {
		esCfg := esrepo.Config{Addresses: cfg.EsAddressesOrDefault(), Index: cfg.ESIndex, Username: cfg.ESUsername, Password: cfg.ESPassword}
		r, err := esrepo.New(esCfg)
		if err != nil {
			log.Printf("failed to init ES repo, falling back to memory: %v", err)
			kbRepo = kb.NewMemoryRepo()
			esInitOK = false
		} else {
			kbRepo = r
			esInitOK = true
			esRepoInstance = r
		}
	} else {
		kbRepo = kb.NewMemoryRepo()
		esInitOK = true
	}

	var h *server.Hertz
	// allow disabling prometheus exporter via env PROM_DISABLE=1|true OR when under go test (to avoid port :9100 conflicts)
	promDisabled := strings.EqualFold(os.Getenv("PROM_DISABLE"), "1") || strings.EqualFold(os.Getenv("PROM_DISABLE"), "true") || isGoTest()
	if !promDisabled {
		promOnce.Do(func() {
			// create first server with tracer
			h = server.Default(server.WithHostPorts(getAddr(cfg)), server.WithTracer(prom.NewServerTracer(":9100", "/metrics", prom.WithEnableGoCollector(true))))
		})
	}
	if h == nil { // subsequent builds without adding tracer to avoid duplicate /metrics
		h = server.Default(server.WithHostPorts(getAddr(cfg)))
	}
	// (middlewares stripped during cleanup – placeholder retained)
	// project headers middleware
	h.Use(func(c context.Context, ctx *app.RequestContext) {
		ctx.Response.Header.Set("X-AssistFusion-Project", common.ProjectName)
		ctx.Response.Header.Set("X-AssistFusion-Version", common.ProjectVersion)
		ctx.Next(c)
	})
	// domain metrics snapshot under separate path to avoid polluting standard prometheus namespace
	h.GET("/metrics/domain", func(c context.Context, ctx *app.RequestContext) {
		ctx.Response.Header.Set(headerContentType, contentTypeTextPlain)
		ctx.Write([]byte(observability.Snapshot()))
	})
	// Health & readiness via new router helper
	router.RegisterHealth(h, func(ctx context.Context) error {
		if esRepoInstance != nil {
			return esRepoInstance.Ping(ctx)
		}
		return nil
	}, cfg.KBBackend == "es", esInitOK)

	if cfg.FeatureRPC {
		ad, err := gateway.NewRPCAdapter(cfg)
		if err != nil {
			log.Printf("failed to init RPC adapter, fallback to local: %v", err)
		} else {
			log.Printf("gateway running in RPC mode")
			kbShim := kbClientShim{c: rpcClients.KBClient}
			aiShim := aiClientShim{c: rpcClients.AIClient}
			router.RegisterKBRPC(h, kbShim)
			router.RegisterAIRPC(h, aiShim)
			router.RegisterUI(h, embeddedUIProviderInstance())
			router.RegisterTicketRPC(h, ad.Ticket)
			return h
		}
	}
	log.Printf("gateway running in LOCAL mode")
	// local routes
	router.RegisterKBLocal(h, kbRepo)
	router.RegisterAILocal(h)
	router.RegisterUI(h, embeddedUIProviderInstance())
	router.RegisterTicketLocal(h, repo)
	return h
}

// embeddedUIProvider bridges old getUIFS() to new router.UIFSProvider
type embeddedUIProvider struct{}

func (embeddedUIProvider) UI() fs.FS { return getUIFS() }

var uiProvOnce sync.Once
var uiProv *embeddedUIProvider

func embeddedUIProviderInstance() *embeddedUIProvider {
	uiProvOnce.Do(func() { uiProv = &embeddedUIProvider{} })
	return uiProv
}

// --- Shims to satisfy router.DepsKB / DepsAI ---
type kbClientShim struct{ c kbservice.Client }

func (s kbClientShim) KBClient() kbservice.Client { return s.c }

type aiClientShim struct{ c aiservice.Client }

func (s aiClientShim) AIClient() aiservice.Client { return s.c }

// httpError: localized HTTP error writer so gateway 不再依赖 common.WriteError。
// 保留最小字段并附加 BizStatusError 便于后续统一日志 / tracing。
func httpError(ctx *app.RequestContext, status int, code, msg string) {
	gwerrors.HTTPError(ctx, status, code, msg)
}

// override registerHealthRoutes to use esInitOK
// legacy ticket helpers removed

// test helper (not exposed in production builds): startTestServer returns server and bound address
func startTestServer(t interface {
	Fatalf(format string, args ...any)
}) (*server.Hertz, string) {
	cfg := common.LoadConfig()
	// choose an ephemeral port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	cfg.HTTPAddr = addr
	// force ES backend if env demands; else memory
	h := BuildServer(cfg)
	go h.Spin()
	return h, addr
}
