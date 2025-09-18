package common

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Middlewares returns the standard middleware chain (recovery, request id, access log).
func Middlewares() []app.HandlerFunc {
	return []app.HandlerFunc{
		recoveryMiddleware(),
		requestIDMiddleware(),
		accessLogMiddleware(),
	}
}

func InitHertzLogger() { /* hook if future Hertz logger init needed */ }

func recoveryMiddleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		defer func() {
			if r := recover(); r != nil {
				if Logger != nil {
					Logger.Error("panic recovered", zap.Any("err", r))
				}
				WriteError(c, ctx, 500, ErrCodeInternal, "internal server error")
			}
		}()
		ctx.Next(c)
	}
}

func requestIDMiddleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.GetHeader("X-Request-ID"))
		if id == "" {
			id = uuid.NewString()
		}
		ctx.Set(RequestIDKey, id)
		ctx.Response.Header.Set("X-Request-ID", id)
		ctx.Next(c)
	}
}

func accessLogMiddleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		start := time.Now()
		ctx.Next(c)
		duration := time.Since(start)
		if Logger != nil {
			Logger.Info("access",
				zap.String("method", string(ctx.Method())),
				zap.String("path", string(ctx.Path())),
				zap.Int("status", ctx.Response.StatusCode()),
				zap.Duration("latency", duration),
			)
		}
	}
}
