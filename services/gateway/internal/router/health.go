package router

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

// RegisterHealth registers /health and /ready endpoints.
// esPing optional: nil-safe func performing backend ping returning error if not ready.
// esPlanned indicates whether ES backend was configured (to differentiate degraded vs pure memory mode).
func RegisterHealth(h *server.Hertz, esPing func(ctx context.Context) error, esPlanned bool, esInitOK bool) {
	h.GET("/health", func(c context.Context, ctx *app.RequestContext) { ctx.JSON(200, map[string]any{"status": "ok"}) })
	h.GET("/ready", func(c context.Context, ctx *app.RequestContext) {
		if esPlanned { // ES selected in config
			if !esInitOK { // init failed earlier
				ctx.JSON(503, map[string]any{"status": "degraded", "kb": "memory-fallback", "es": "init-failed"})
				return
			}
			if esPing != nil {
				pingCtx, cancel := context.WithTimeout(c, 400*time.Millisecond)
				defer cancel()
				if err := esPing(pingCtx); err != nil {
					ctx.JSON(503, map[string]any{"status": "degraded", "kb": "memory-fallback", "es": "ping-failed", "error": err.Error()})
					return
				}
			}
			ctx.JSON(200, map[string]any{"status": "ready", "backend": "es"})
			return
		}
		ctx.JSON(200, map[string]any{"status": "ready", "backend": "memory"})
	})
}
