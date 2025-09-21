package observability

import (
	"context"

	"github.com/cloudwego/kitex/pkg/klog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// InitTracing sets up a basic stdout tracer provider (development only).
// Returns a shutdown func to flush spans.
func InitTracing(service string) (func(context.Context) error, error) {
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(service),
		)),
	)
	klog.Infof("tracing initialized service=%s exporter=none", service)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
