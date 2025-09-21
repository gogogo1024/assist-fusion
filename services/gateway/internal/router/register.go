package router

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/gogogo1024/assist-fusion/internal/kb"
)

// Deps aggregates dependencies for route registration.
// In future can include logger, metrics, middlewares etc.
type Deps struct {
	UseRPC bool

	// Local repos (when not using RPC)
	TicketRepo interface{} // kept as interface{} for now; replace with concrete common.TicketRepo if exported
	KBRepo     kb.Repo

	// RPC clients (when UseRPC)
	TicketsRPC interface { /* placeholder for ticket client accessor */
	}
	KBRPC DepsKB
	AIRPC DepsAI

	// UI provider
	UI UIFSProvider
}

// RegisterAll wires every route group based on Deps.
func RegisterAll(h *server.Hertz, d Deps) {
	RegisterHealth(h) // health always

	// Tickets (already split local/rpc via separate functions)
	if d.UseRPC {
		if tr, ok := d.TicketsRPC.(interface{ Register(h *server.Hertz) }); ok { // placeholder until ticket RPC deps struct
			tr.Register(h)
		}
	} else {
		if repo, ok := d.TicketRepo.(interface { /* marker */
		}); ok {
			_ = repo // ticket registration already done outside pending refactor
		}
	}

	// KB
	if d.UseRPC && d.KBRPC != nil {
		RegisterKBRPC(h, d.KBRPC)
	} else if d.KBRepo != nil {
		RegisterKBLocal(h, d.KBRepo)
	}

	// AI
	if d.UseRPC && d.AIRPC != nil {
		RegisterAIRPC(h, d.AIRPC)
	} else {
		RegisterAILocal(h)
	}

	// UI
	if d.UI != nil {
		RegisterUI(h, d.UI)
	}
}
