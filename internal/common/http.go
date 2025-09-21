package common

import (
	"context"
	"log"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

// Basic project metadata (was previously in removed files)
const (
	ProjectName    = "assist-fusion"
	ProjectVersion = "0.1.0"
)

// InitLogger provides a minimal std logger init (placeholder for structured logger).
func InitLogger() { /* no-op placeholder; replace with zap or similar if needed */ }

// InitHertzLogger sets hlog to use std log output for simplicity.
func InitHertzLogger() { hlog.SetOutput(log.Writer()) }

// Middlewares returns a slice of Hertz middleware (kept minimal after cleanup).
// Middlewares returns an empty list (placeholder to keep previous call sites compiling).
type HertzMiddleware func(ctx context.Context, c *app.RequestContext)

func Middlewares() []HertzMiddleware { return nil }

// (error helpers retained in error.go)
