package mtl

import (
	"net"
	"net/http"
	"strings"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/registry"
	"github.com/cloudwego/kitex/server"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	consul "github.com/kitex-contrib/registry-consul"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var Registry *prometheus.Registry

// InitMetrics mirrors the external common implementation: create a dedicated
// prometheus Registry, register go & process collectors, plus existing RPC
// collectors (observability). It optionally registers a pseudo service named
// "prometheus" into consul (tagged with the real service name) when a
// registryAddr is provided, and exposes /metrics on the given metricsPort.
func InitMetrics(serviceName, metricsPort, registryAddr string) (registry.Registry, *registry.Info) {
	if metricsPort == "" { // disabled
		return nil, nil
	}
	listenAddr := metricsPort
	if !strings.Contains(listenAddr, ":") {
		listenAddr = ":" + listenAddr
	}
	Registry = prometheus.NewRegistry()
	Registry.MustRegister(collectors.NewGoCollector())
	Registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	// integrate existing RPC metrics
	observability.RegisterCollectors(Registry)

	var r registry.Registry
	var info *registry.Info

	// consul registration (optional)
	if registryAddr != "" {
		if reg, err := consul.NewConsulRegister(registryAddr); err != nil {
			klog.Warnf("consul register (metrics) init failed: %v", err)
		} else {
			// resolve addr for registry info (use host:port as given)
			tcpAddr, err := net.ResolveTCPAddr("tcp", listenAddr)
			if err != nil {
				klog.Warnf("resolve metrics addr failed: %v", err)
			} else {
				info = &registry.Info{ServiceName: "prometheus", Addr: tcpAddr, Weight: 1, Tags: map[string]string{"service": serviceName}}
				if err = reg.Register(info); err != nil {
					klog.Warnf("consul register metrics failed: %v", err)
				} else {
					r = reg
					server.RegisterShutdownHook(func() {
						_ = reg.Deregister(info)
					})
				}
			}
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(Registry, promhttp.HandlerOpts{}))
	go func() {
		if err := http.ListenAndServe(listenAddr, mux); err != nil && !strings.Contains(err.Error(), "Server closed") {
			klog.Errorf("metrics http server error: %v", err)
		}
	}()
	klog.Infof("metrics server listening on %s service=%s consul_registered=%v", listenAddr, serviceName, info != nil)
	return r, info
}
