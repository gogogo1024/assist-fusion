package router

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"net/http"

	"github.com/gogogo1024/assist-fusion/internal/gateway"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	kcommon "github.com/gogogo1024/assist-fusion/kitex_gen/common"
	gwerrors "github.com/gogogo1024/assist-fusion/services/gateway/internal/errors"
)

// RegisterTicketRPC registers ticket routes backed by RPC adapter.
func RegisterTicketRPC(h *server.Hertz, api gateway.TicketAPI) {
	registerTicketCRUD(h, api)
	registerTicketActions(h, api)
	registerTicketMeta(h, api)
}

// registerTicketCRUD sets up create/list/get endpoints.
func registerTicketCRUD(h *server.Hertz, api gateway.TicketAPI) {
	h.POST(PathTickets, func(c context.Context, ctx *app.RequestContext) {
		var req struct {
			Title string `json:"title"`
			Desc  string `json:"desc"`
			Note  string `json:"note"`
		}
		if err := ctx.Bind(&req); err != nil || req.Title == "" {
			gwerrors.HTTPError(ctx, http.StatusBadRequest, "bad_request", gwerrors.MsgBadRequest)
			return
		}
		t, err := api.Create(c, req.Title, req.Desc, req.Note)
		if err != nil {
			gwerrors.MapServiceError(ctx, err)
			return
		}
		observability.TicketCreated.Add(1)
		ctx.JSON(201, normalizeTicket(t))
	})
	h.GET(PathTickets, func(c context.Context, ctx *app.RequestContext) {
		ts, err := api.List(c)
		if err != nil {
			gwerrors.MapServiceError(ctx, err)
			return
		}
		out := make([]any, 0, len(ts))
		for _, t := range ts {
			out = append(out, normalizeTicket(t))
		}
		ctx.JSON(200, out)
	})
	h.GET(PathTicketID, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, err := api.Get(c, id)
		if err != nil {
			gwerrors.MapServiceError(ctx, err)
			return
		}
		if t == nil {
			gwerrors.HTTPError(ctx, http.StatusNotFound, "not_found", gwerrors.MsgNotFound)
			return
		}
		ctx.JSON(200, normalizeTicket(t))
	})
}

// registerTicketActions sets up action endpoints (assign/resolve/escalate/reopen).
func registerTicketActions(h *server.Hertz, api gateway.TicketAPI) {
	h.PUT(PathTicketAssign, func(c context.Context, ctx *app.RequestContext) { ticketActionRPC(c, ctx, api.Assign) })
	h.PUT(PathTicketResolve, func(c context.Context, ctx *app.RequestContext) { ticketActionRPC(c, ctx, api.Resolve) })
	h.PUT(PathTicketEscalate, func(c context.Context, ctx *app.RequestContext) { ticketActionRPC(c, ctx, api.Escalate) })
	h.PUT(PathTicketReopen, func(c context.Context, ctx *app.RequestContext) { ticketActionRPC(c, ctx, api.Reopen) })
}

// registerTicketMeta sets up informational endpoints (cycles / events)
func registerTicketMeta(h *server.Hertz, api gateway.TicketAPI) {
	h.GET(PathTicketCycles, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		cs, err := api.Cycles(c, id)
		if err != nil {
			gwerrors.MapServiceError(ctx, err)
			return
		}
		// optional authoritative fetch (ignore error; fallback below)
		var currentFromTicket *int32
		if tkt, gerr := api.Get(c, id); gerr == nil && tkt != nil && int(tkt.CurrentCycle) < len(cs) {
			currentFromTicket = &tkt.CurrentCycle
		}
		cyclesOut := make([]map[string]any, 0, len(cs))
		for _, cy := range cs {
			cyclesOut = append(cyclesOut, map[string]any{
				"created_at":   cy.CreatedAt,
				"assigned_at":  cy.AssignedAt,
				"resolved_at":  cy.ResolvedAt,
				"escalated_at": cy.EscalatedAt,
				"status":       strings.ToLower(cy.Status.String()),
			})
		}
		current := 0
		if currentFromTicket != nil {
			current = int(*currentFromTicket)
		} else if n := len(cyclesOut); n > 0 { // fallback heuristic
			current = n - 1
		}
		ctx.JSON(200, map[string]any{"current": current, "cycles": cyclesOut})
	})
	h.GET(PathTicketEvents, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		es, err := api.Events(c, id)
		if err != nil {
			gwerrors.MapServiceError(ctx, err)
			return
		}
		ctx.JSON(200, map[string]any{"events": es})
	})
}

func ticketActionRPC(c context.Context, ctx *app.RequestContext, fn func(context.Context, string, string) (*kcommon.Ticket, error)) {
	id := string(ctx.Param("id"))
	var req struct {
		Note string `json:"note"`
	}
	if b := ctx.Request.Body(); len(b) > 0 {
		_ = ctx.Bind(&req)
	}
	t, err := fn(c, id, req.Note)
	if err != nil {
		gwerrors.MapServiceError(ctx, err)
		return
	}
	ctx.JSON(200, normalizeTicket(t))
}

// normalizeTicket converts thrift enum TicketStatus (numbers) to expected lowercase strings for HTTP clients.
func normalizeTicket(t *kcommon.Ticket) *struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Desc         string `json:"desc"`
	Status       string `json:"status"`
	CreatedAt    int64  `json:"created_at"`
	AssignedAt   int64  `json:"assigned_at"`
	ResolvedAt   int64  `json:"resolved_at"`
	EscalatedAt  int64  `json:"escalated_at"`
	ReopenedAt   int64  `json:"reopened_at"`
	CurrentCycle int32  `json:"current_cycle"`
	Cycles       []*struct {
		CreatedAt   int64  `json:"created_at"`
		AssignedAt  int64  `json:"assigned_at"`
		ResolvedAt  int64  `json:"resolved_at"`
		EscalatedAt int64  `json:"escalated_at"`
		Status      string `json:"status"`
	} `json:"cycles,omitempty"`
	Events []*struct {
		Type string `json:"type"`
		At   int64  `json:"at"`
		Note string `json:"note"`
	} `json:"events,omitempty"`
} {
	if t == nil {
		return nil
	}
	status := strings.ToLower(t.Status.String())
	cycles := make([]*struct {
		CreatedAt   int64  `json:"created_at"`
		AssignedAt  int64  `json:"assigned_at"`
		ResolvedAt  int64  `json:"resolved_at"`
		EscalatedAt int64  `json:"escalated_at"`
		Status      string `json:"status"`
	}, 0, len(t.Cycles))
	for _, c := range t.Cycles {
		cycles = append(cycles, &struct {
			CreatedAt   int64  `json:"created_at"`
			AssignedAt  int64  `json:"assigned_at"`
			ResolvedAt  int64  `json:"resolved_at"`
			EscalatedAt int64  `json:"escalated_at"`
			Status      string `json:"status"`
		}{
			CreatedAt:   c.CreatedAt,
			AssignedAt:  c.AssignedAt,
			ResolvedAt:  c.ResolvedAt,
			EscalatedAt: c.EscalatedAt,
			Status:      strings.ToLower(c.Status.String()),
		})
	}
	events := make([]*struct {
		Type string `json:"type"`
		At   int64  `json:"at"`
		Note string `json:"note"`
	}, 0, len(t.Events))
	for _, e := range t.Events {
		events = append(events, &struct {
			Type string `json:"type"`
			At   int64  `json:"at"`
			Note string `json:"note"`
		}{Type: e.Type, At: e.At, Note: e.Note})
	}
	return &struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		Desc         string `json:"desc"`
		Status       string `json:"status"`
		CreatedAt    int64  `json:"created_at"`
		AssignedAt   int64  `json:"assigned_at"`
		ResolvedAt   int64  `json:"resolved_at"`
		EscalatedAt  int64  `json:"escalated_at"`
		ReopenedAt   int64  `json:"reopened_at"`
		CurrentCycle int32  `json:"current_cycle"`
		Cycles       []*struct {
			CreatedAt   int64  `json:"created_at"`
			AssignedAt  int64  `json:"assigned_at"`
			ResolvedAt  int64  `json:"resolved_at"`
			EscalatedAt int64  `json:"escalated_at"`
			Status      string `json:"status"`
		} `json:"cycles,omitempty"`
		Events []*struct {
			Type string `json:"type"`
			At   int64  `json:"at"`
			Note string `json:"note"`
		} `json:"events,omitempty"`
	}{
		ID:           t.Id,
		Title:        t.Title,
		Desc:         t.Desc,
		Status:       status,
		CreatedAt:    t.CreatedAt,
		AssignedAt:   t.AssignedAt,
		ResolvedAt:   t.ResolvedAt,
		EscalatedAt:  t.EscalatedAt,
		ReopenedAt:   t.ReopenedAt,
		CurrentCycle: t.CurrentCycle,
		Cycles:       cycles,
		Events:       events,
	}
}
