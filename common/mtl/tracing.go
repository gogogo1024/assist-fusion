package mtl

import (
	"context"

	provider "github.com/kitex-contrib/obs-opentelemetry/provider"
)

// InitTracing initializes OpenTelemetry provider (metrics disabled here because we use custom registry)
// and returns an object exposing Shutdown(context.Context) error.
func InitTracing(serviceName string) interface{ Shutdown(context.Context) error } {
	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName(serviceName),
		provider.WithInsecure(),
		provider.WithEnableMetrics(false),
	)
	return p
}
