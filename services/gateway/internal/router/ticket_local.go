package router

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/google/uuid"

	"github.com/gogogo1024/assist-fusion/internal/common"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	gwerrors "github.com/gogogo1024/assist-fusion/services/gateway/internal/errors"
)

// RegisterTicketLocal registers ticket routes using a local in-memory repository.
func RegisterTicketLocal(h *server.Hertz, repo common.TicketRepo) {
	h.POST(PathTickets, func(c context.Context, ctx *app.RequestContext) {
		var req struct {
			Title string `json:"title"`
			Desc  string `json:"desc"`
			Note  string `json:"note"`
		}
		if err := ctx.Bind(&req); err != nil {
			gwerrors.HTTPError(ctx, 400, common.ErrCodeBadRequest, gwerrors.MsgBadRequest)
			return
		}
		now := time.Now().Unix()
		t := &common.Ticket{ID: uuid.NewString(), Title: req.Title, Desc: req.Desc, Status: "created", CreatedAt: now, Cycles: []common.TicketCycle{{CreatedAt: now, Status: "created"}}, CurrentCycle: 0, Events: []common.TicketEvent{{Type: "created", At: now, Note: req.Note}}}
		repo.Create(c, t)
		observability.TicketCreated.Add(1)
		ctx.JSON(201, t)
	})
	// list
	h.GET(PathTickets, func(c context.Context, ctx *app.RequestContext) { ts, _ := repo.List(c); ctx.JSON(200, ts) })

	// get
	h.GET(PathTicketID, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, _ := repo.Get(c, id)
		if t == nil {
			gwerrors.HTTPError(ctx, 404, common.ErrCodeNotFound, gwerrors.MsgNotFound)
			return
		}
		ctx.JSON(200, t)
	})

	// assign
	h.PUT(PathTicketAssign, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, _ := repo.Get(c, id)
		if t == nil {
			gwerrors.HTTPError(ctx, 404, common.ErrCodeNotFound, gwerrors.MsgNotFound)
			return
		}
		var req struct {
			Note     string `json:"note"`
			Assignee string `json:"assignee"`
		}
		if b := ctx.Request.Body(); len(b) > 0 {
			_ = ctx.Bind(&req)
		}
		now := time.Now().Unix()
		t.AssignedAt = now
		if req.Assignee != "" {
			t.Assignee = req.Assignee
		}
		t.Status = "assigned"
		if t.CurrentCycle >= 0 && t.CurrentCycle < len(t.Cycles) {
			cyc := &t.Cycles[t.CurrentCycle]
			cyc.AssignedAt = now
			cyc.Status = "assigned"
		}
		t.Events = append(t.Events, common.TicketEvent{Type: "assigned", At: now, Note: req.Note})
		repo.Update(c, t)
		observability.TicketAssigned.Add(1)
		ctx.JSON(200, t)
	})
	// resolve
	h.PUT(PathTicketResolve, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, _ := repo.Get(c, id)
		if t == nil {
			gwerrors.HTTPError(ctx, 404, common.ErrCodeNotFound, gwerrors.MsgNotFound)
			return
		}
		var req struct {
			Note string `json:"note"`
		}
		if b := ctx.Request.Body(); len(b) > 0 {
			_ = ctx.Bind(&req)
		}
		now := time.Now().Unix()
		t.ResolvedAt = now
		t.Status = "resolved"
		if t.CurrentCycle >= 0 && t.CurrentCycle < len(t.Cycles) {
			cyc := &t.Cycles[t.CurrentCycle]
			cyc.ResolvedAt = now
			cyc.Status = "resolved"
		}
		t.Events = append(t.Events, common.TicketEvent{Type: "resolved", At: now, Note: req.Note})
		repo.Update(c, t)
		observability.TicketResolved.Add(1)
		ctx.JSON(200, t)
	})
	// escalate
	h.PUT(PathTicketEscalate, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, _ := repo.Get(c, id)
		if t == nil {
			gwerrors.HTTPError(ctx, 404, common.ErrCodeNotFound, gwerrors.MsgNotFound)
			return
		}
		if t.Status == "resolved" || t.Status == "closed" || t.Status == "canceled" {
			gwerrors.HTTPError(ctx, 409, common.ErrCodeConflict, "cannot escalate resolved ticket")
			return
		}
		var req struct {
			Note string `json:"note"`
		}
		if b := ctx.Request.Body(); len(b) > 0 {
			_ = ctx.Bind(&req)
		}
		now := time.Now().Unix()
		t.EscalatedAt = now
		t.Status = "escalated"
		if t.CurrentCycle >= 0 && t.CurrentCycle < len(t.Cycles) {
			cyc := &t.Cycles[t.CurrentCycle]
			cyc.EscalatedAt = now
			cyc.Status = "escalated"
		}
		t.Events = append(t.Events, common.TicketEvent{Type: "escalated", At: now, Note: req.Note})
		repo.Update(c, t)
		observability.TicketEscalated.Add(1)
		ctx.JSON(200, t)
	})
	// start
	h.PUT(PathTicketStart, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, _ := repo.Get(c, id)
		if t == nil {
			gwerrors.HTTPError(ctx, 404, common.ErrCodeNotFound, gwerrors.MsgNotFound)
			return
		}
		if t.Status == "resolved" || t.Status == "closed" || t.Status == "canceled" {
			gwerrors.HTTPError(ctx, 409, common.ErrCodeConflict, "cannot start terminal ticket")
			return
		}
		var req struct {
			Note string `json:"note"`
		}
		if b := ctx.Request.Body(); len(b) > 0 {
			_ = ctx.Bind(&req)
		}
		now := time.Now().Unix()
		t.Status = "in_progress"
		if t.CurrentCycle >= 0 && t.CurrentCycle < len(t.Cycles) {
			cyc := &t.Cycles[t.CurrentCycle]
			cyc.Status = "in_progress"
		}
		t.Events = append(t.Events, common.TicketEvent{Type: "started", At: now, Note: req.Note})
		repo.Update(c, t)
		ctx.JSON(200, t)
	})
	// wait
	h.PUT(PathTicketWait, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, _ := repo.Get(c, id)
		if t == nil {
			gwerrors.HTTPError(ctx, 404, common.ErrCodeNotFound, gwerrors.MsgNotFound)
			return
		}
		if t.Status == "resolved" || t.Status == "closed" || t.Status == "canceled" {
			gwerrors.HTTPError(ctx, 409, common.ErrCodeConflict, "cannot wait terminal ticket")
			return
		}
		var req struct {
			Note string `json:"note"`
		}
		if b := ctx.Request.Body(); len(b) > 0 {
			_ = ctx.Bind(&req)
		}
		now := time.Now().Unix()
		t.Status = "waiting"
		if t.CurrentCycle >= 0 && t.CurrentCycle < len(t.Cycles) {
			cyc := &t.Cycles[t.CurrentCycle]
			cyc.Status = "waiting"
		}
		t.Events = append(t.Events, common.TicketEvent{Type: "waiting", At: now, Note: req.Note})
		repo.Update(c, t)
		ctx.JSON(200, t)
	})
	// close
	h.PUT(PathTicketClose, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, _ := repo.Get(c, id)
		if t == nil {
			gwerrors.HTTPError(ctx, 404, common.ErrCodeNotFound, gwerrors.MsgNotFound)
			return
		}
		if t.Status == "closed" || t.Status == "canceled" {
			gwerrors.HTTPError(ctx, 409, common.ErrCodeConflict, "ticket already terminal")
			return
		}
		var req struct {
			Note string `json:"note"`
		}
		if b := ctx.Request.Body(); len(b) > 0 {
			_ = ctx.Bind(&req)
		}
		now := time.Now().Unix()
		t.ClosedAt = now
		t.Status = "closed"
		if t.CurrentCycle >= 0 && t.CurrentCycle < len(t.Cycles) {
			cyc := &t.Cycles[t.CurrentCycle]
			cyc.ClosedAt = now
			cyc.Status = "closed"
		}
		t.Events = append(t.Events, common.TicketEvent{Type: "closed", At: now, Note: req.Note})
		repo.Update(c, t)
		ctx.JSON(200, t)
	})
	// cancel
	h.PUT(PathTicketCancel, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, _ := repo.Get(c, id)
		if t == nil {
			gwerrors.HTTPError(ctx, 404, common.ErrCodeNotFound, gwerrors.MsgNotFound)
			return
		}
		if t.Status == "closed" || t.Status == "canceled" {
			gwerrors.HTTPError(ctx, 409, common.ErrCodeConflict, "ticket already terminal")
			return
		}
		var req struct {
			Note string `json:"note"`
		}
		if b := ctx.Request.Body(); len(b) > 0 {
			_ = ctx.Bind(&req)
		}
		now := time.Now().Unix()
		t.CanceledAt = now
		t.Status = "canceled"
		if t.CurrentCycle >= 0 && t.CurrentCycle < len(t.Cycles) {
			cyc := &t.Cycles[t.CurrentCycle]
			cyc.CanceledAt = now
			cyc.Status = "canceled"
		}
		t.Events = append(t.Events, common.TicketEvent{Type: "canceled", At: now, Note: req.Note})
		repo.Update(c, t)
		ctx.JSON(200, t)
	})
	// reopen
	h.PUT(PathTicketReopen, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, _ := repo.Get(c, id)
		if t == nil {
			gwerrors.HTTPError(ctx, 404, common.ErrCodeNotFound, gwerrors.MsgNotFound)
			return
		}
		if t.Status != "resolved" {
			gwerrors.HTTPError(ctx, 409, common.ErrCodeConflict, "can only reopen resolved ticket")
			return
		}
		var req struct {
			Note string `json:"note"`
		}
		if b := ctx.Request.Body(); len(b) > 0 {
			_ = ctx.Bind(&req)
		}
		now := time.Now().Unix()
		t.ReopenedAt = now
		t.Cycles = append(t.Cycles, common.TicketCycle{CreatedAt: now, Status: "created"})
		t.CurrentCycle = len(t.Cycles) - 1
		t.Status = "created"
		t.AssignedAt, t.ResolvedAt, t.EscalatedAt = 0, 0, 0
		t.Events = append(t.Events, common.TicketEvent{Type: "reopened", At: now, Note: req.Note})
		repo.Update(c, t)
		observability.TicketReopened.Add(1)
		ctx.JSON(200, t)
	})
	// cycles
	h.GET(PathTicketCycles, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, _ := repo.Get(c, id)
		if t == nil {
			gwerrors.HTTPError(ctx, 404, common.ErrCodeNotFound, gwerrors.MsgNotFound)
			return
		}
		ctx.JSON(200, map[string]any{"current": t.CurrentCycle, "cycles": t.Cycles})
	})
	// events
	h.GET(PathTicketEvents, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, _ := repo.Get(c, id)
		if t == nil {
			gwerrors.HTTPError(ctx, 404, common.ErrCodeNotFound, gwerrors.MsgNotFound)
			return
		}
		ctx.JSON(200, map[string]any{"events": t.Events})
	})
}
