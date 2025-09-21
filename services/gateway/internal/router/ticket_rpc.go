package router

import (
	"context"

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
		ctx.JSON(201, t)
	})
	h.GET(PathTickets, func(c context.Context, ctx *app.RequestContext) {
		ts, err := api.List(c)
		if err != nil {
			gwerrors.MapServiceError(ctx, err)
			return
		}
		ctx.JSON(200, ts)
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
		ctx.JSON(200, t)
	})
	// actions
	h.PUT(PathTicketAssign, func(c context.Context, ctx *app.RequestContext) { ticketActionRPC(c, ctx, api.Assign) })
	h.PUT(PathTicketResolve, func(c context.Context, ctx *app.RequestContext) { ticketActionRPC(c, ctx, api.Resolve) })
	h.PUT(PathTicketEscalate, func(c context.Context, ctx *app.RequestContext) { ticketActionRPC(c, ctx, api.Escalate) })
	h.PUT(PathTicketReopen, func(c context.Context, ctx *app.RequestContext) { ticketActionRPC(c, ctx, api.Reopen) })
	h.GET(PathTicketCycles, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		cs, err := api.Cycles(c, id)
		if err != nil {
			gwerrors.MapServiceError(ctx, err)
			return
		}
		ctx.JSON(200, map[string]any{"cycles": cs})
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
	ctx.JSON(200, t)
}
