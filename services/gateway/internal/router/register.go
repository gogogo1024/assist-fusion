package router

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app/server"
)

// Deps aggregates dependencies for route registration.
// In future can include logger, metrics, middlewares etc.
type Deps struct {
	TicketsRPC interface { /* placeholder for ticket client accessor */
	}
	KBRPC DepsKB
	AIRPC DepsAI
	UI    UIFSProvider
}

// RegisterAll wires every route group based on Deps.
func RegisterAll(h *server.Hertz, d Deps) {
	RegisterHealth(h, func(ctx context.Context) error { return nil }, false, true)
	if tr, ok := d.TicketsRPC.(interface{ Register(h *server.Hertz) }); ok {
		tr.Register(h)
	}
	if d.KBRPC != nil {
		RegisterKBRPC(h, d.KBRPC)
	}
	if d.AIRPC != nil {
		RegisterAIRPC(h, d.AIRPC)
	}
	if d.UI != nil {
		RegisterUI(h, d.UI)
	}
}
