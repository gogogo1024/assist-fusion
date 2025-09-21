package observability

import (
	"context"
	"net/http"
	"time"

	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/prometheus/client_golang/prometheus"
	promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	reqCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kitex",
			Name:      "requests_total",
			Help:      "Total RPC requests",
		},
		[]string{"service", "method", "status"},
	)
	reqLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "kitex",
			Name:      "request_duration_seconds",
			Help:      "RPC request latency in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"service", "method"},
	)
	metricsRegistered bool
)

// RegisterCollectors allows external registries (e.g., common/mtl) to reuse
// the same metric vectors instead of duplicating definitions. If a registry
// is provided and collectors are not yet registered, it registers them there;
// otherwise it falls back to the default global registry.
func RegisterCollectors(reg *prometheus.Registry) {
	if metricsRegistered {
		return
	}
	if reg != nil {
		reg.MustRegister(reqCounter, reqLatency)
	} else {
		prometheus.MustRegister(reqCounter, reqLatency)
	}
	metricsRegistered = true
}

// InitMetrics launches a /metrics HTTP endpoint if addr not empty.
func InitMetrics(service, addr string) *http.Server {
	if addr == "" {
		return nil
	}
	if !metricsRegistered {
		prometheus.MustRegister(reqCounter, reqLatency)
		metricsRegistered = true
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Errorf("metrics server error: %v", err)
		}
	}()
	klog.Infof("metrics server listening on %s service=%s", addr, service)
	return srv
}

// Middleware collects count & latency metrics for each RPC call.
func Middleware(service string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req, resp interface{}) (err error) {
			start := time.Now()
			err = next(ctx, req, resp)
			ri := rpcinfo.GetRPCInfo(ctx)
			method := "unknown"
			if ri != nil && ri.Invocation() != nil {
				method = ri.Invocation().MethodName()
			}
			status := "ok"
			if err != nil {
				status = "error"
			}
			reqCounter.WithLabelValues(service, method, status).Inc()
			reqLatency.WithLabelValues(service, method).Observe(time.Since(start).Seconds())
			return err
		}
	}
}
