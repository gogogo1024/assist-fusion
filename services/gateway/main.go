package main

import (
	"context"
	"io/fs"
	"log"
	"net"
	"os"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	prom "github.com/hertz-contrib/monitor-prometheus"

	"github.com/gogogo1024/assist-fusion/internal/common"
	"github.com/gogogo1024/assist-fusion/internal/gateway"
	rpcClients "github.com/gogogo1024/assist-fusion/internal/gateway/rpc"
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

// BuildServer assembles the Hertz server with all routes for reuse in tests.
func BuildServer(cfg *common.Config) *server.Hertz {
	common.InitLogger()
	common.InitHertzLogger()
	// Local in-memory mode removed: gateway now always requires RPC backends.
	// ES repository initialization for gateway health removed (responsibility moved to kb-rpc service). Keep flags neutral.
	esInitOK = true

	var h *server.Hertz
	promOnce.Do(func() {
		// always enable prometheus exporter
		h = server.Default(server.WithHostPorts(getAddr(cfg)), server.WithTracer(prom.NewServerTracer(":9100", "/metrics", prom.WithEnableGoCollector(true))))
	})
	if h == nil {
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

	ad, err := gateway.NewRPCAdapter(cfg)
	if err != nil {
		// Hard failure now (no fallback to removed local mode)
		log.Fatalf("failed to init RPC adapter (local mode removed): %v", err)
	}
	log.Printf("gateway running in RPC mode (local mode removed)")
	kbShim := kbClientShim{c: rpcClients.KBClient}
	aiShim := aiClientShim{c: rpcClients.AIClient}
	router.RegisterKBRPC(h, kbShim)
	router.RegisterAIRPC(h, aiShim)
	// enable vector search endpoint (best-effort; relies on AI embeddings)
	router.EnableVectorSearch(h, rpcClients.AIClient)
	router.RegisterUI(h, embeddedUIProviderInstance())
	router.RegisterTicketRPC(h, ad.Ticket)
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
