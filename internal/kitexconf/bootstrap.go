package kitexconf

import (
	"context"
	"fmt"
	"net"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	"github.com/gogogo1024/assist-fusion/common/mtl"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	consul "github.com/kitex-contrib/registry-consul"
)

// BuildServerOptions constructs common server.Options referencing the cart service style.
// It resolves address, sets basic info, and (optionally) attaches a consul registry
// when a registry address is present in configuration.
func BuildServerOptions(cfg *Config) ([]server.Option, error) {
	var opts []server.Option
	if cfg == nil {
		return opts, fmt.Errorf("nil config")
	}
	// resolve listen address
	addr, err := net.ResolveTCPAddr("tcp", cfg.Kitex.Address)
	if err != nil {
		return nil, err
	}
	opts = append(opts, server.WithServiceAddr(addr))
	opts = append(opts, server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: cfg.Kitex.Service}))
	// consul registry (optional) if address provided
	if len(cfg.Registry.RegistryAddress) > 0 {
		r, err := consul.NewConsulRegister(cfg.Registry.RegistryAddress[0])
		if err != nil {
			klog.Warnf("consul registry init failed: %v", err)
		} else {
			opts = append(opts, server.WithRegistry(r))
		}
	}
	// always include RPC metrics middleware (collectors registered later)
	opts = append(opts, server.WithMiddleware(observability.Middleware(cfg.Kitex.Service)))
	return opts, nil
}

// Placeholder for future metrics & tracing integration.
// In cart example they initialize metrics & tracing and deregister on exit.
// We expose a simple hook structure for later extension.

type RuntimeHooks struct{ Shutdown func(context.Context) }

// InitRuntime wires metrics HTTP & tracing (stdout exporter) if configured.
func InitRuntime(ctx context.Context, cfg *Config) *RuntimeHooks {
	var shutdowns []func(context.Context) error
	var deregInfo *mtlDeregister // remains nil until registry restored

	// metrics + consul registration using new mtl package
	if cfg.Kitex.MetricsPort != "" {
		var regAddr string
		if len(cfg.Registry.RegistryAddress) > 0 {
			regAddr = cfg.Registry.RegistryAddress[0]
		}
		r, info := mtl.InitMetrics(cfg.Kitex.Service, cfg.Kitex.MetricsPort, regAddr)
		if r != nil && info != nil {
			deregInfo = &mtlDeregister{}
			shutdowns = append(shutdowns, func(c context.Context) error {
				return r.Deregister(info)
			})
		}
	}
	// tracing via mtl (provider) fallback to legacy if needed
	var tracingClosed bool
	if p := mtl.InitTracing(cfg.Kitex.Service); p != nil {
		shutdowns = append(shutdowns, func(c context.Context) error {
			p.Shutdown(c) // provider has Shutdown(ctx context) signature
			tracingClosed = true
			return nil
		})
	}
	if !tracingClosed { // ensure at least legacy for stdout dev fallback
		if closer, err := observability.InitTracing(cfg.Kitex.Service); err == nil && closer != nil {
			shutdowns = append(shutdowns, closer)
		} else if err != nil {
			klog.Warnf("fallback tracing init failed: %v", err)
		}
	}
	klog.Infof("runtime init complete service=%s env=%s metricsPort=%s deregister=%v", cfg.Kitex.Service, cfg.Env, cfg.Kitex.MetricsPort, deregInfo != nil)
	return &RuntimeHooks{Shutdown: func(c context.Context) {
		for _, fn := range shutdowns {
			_ = fn(c)
		}
	}}
}

type mtlDeregister struct{}
