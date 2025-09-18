package main

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gogogo1024/assist-fusion/internal/ai"
	"github.com/gogogo1024/assist-fusion/internal/common"

	// httpx removed: consolidated error handling into common
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/gogogo1024/assist-fusion/internal/gateway"
	"github.com/gogogo1024/assist-fusion/internal/kb"
	esrepo "github.com/gogogo1024/assist-fusion/internal/kb/esrepo"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	kcommon "github.com/gogogo1024/assist-fusion/kitex_gen/common"
	"github.com/google/uuid"
	prom "github.com/hertz-contrib/monitor-prometheus"
)

const notFoundMsg = "not found"
const badRequestMsg = "bad request"
const kbUnavailableMsg = "kb backend unavailable"
const internalErrMsg = "internal error"

// path constants (avoid duplication)
const (
	pathTickets      = "/v1/tickets"
	pathTicketID     = "/v1/tickets/:id"
	pathDocs         = "/v1/docs"
	pathDocID        = "/v1/docs/:id"
	pathSearch       = "/v1/search"
	pathKBInfo       = "/v1/kb/info"
	pathEmbeddings   = "/v1/embeddings"
	pathTicketCycles = "/v1/tickets/:id/cycles"
	pathTicketEvents = "/v1/tickets/:id/events"
)

var esInitOK bool
var esRepoInstance interface{ Ping(context.Context) error } // optional stored when using ES
var promOnce sync.Once
var prometheusTracerEnabled bool

func main() {
	cfg := common.LoadConfig()
	h := BuildServer(cfg)
	log.Printf("ticket-svc listening on %s", getAddr(cfg))
	h.Spin()
}

func getAddr(cfg *common.Config) string {
	if cfg.HTTPAddr != "" {
		return cfg.HTTPAddr
	}
	if v := os.Getenv("TICKET_ADDR"); v != "" {
		return v
	}
	return ":8081"
}

// BuildServer assembles the Hertz server with all routes for reuse in tests.
func BuildServer(cfg *common.Config) *server.Hertz {
	common.InitLogger()
	common.InitHertzLogger()
	repo := common.NewMemoryTicketRepo()
	var kbRepo kb.Repo
	if cfg.KBBackend == "es" {
		esCfg := esrepo.Config{Addresses: cfg.EsAddressesOrDefault(), Index: cfg.ESIndex, Username: cfg.ESUsername, Password: cfg.ESPassword}
		r, err := esrepo.New(esCfg)
		if err != nil {
			log.Printf("failed to init ES repo, falling back to memory: %v", err)
			kbRepo = kb.NewMemoryRepo()
			esInitOK = false
		} else {
			kbRepo = r
			esInitOK = true
			esRepoInstance = r
		}
	} else {
		kbRepo = kb.NewMemoryRepo()
		esInitOK = true
	}

	var h *server.Hertz
	promOnce.Do(func() {
		// create first server; allow disabling prometheus tracer via env for tests
		if os.Getenv("PROM_DISABLE") == "1" {
			h = server.Default(server.WithHostPorts(getAddr(cfg)))
		} else {
			h = server.Default(
				server.WithHostPorts(getAddr(cfg)),
				server.WithTracer(prom.NewServerTracer(":9100", "/metrics", prom.WithEnableGoCollector(true))),
			)
			prometheusTracerEnabled = true
		}
	})
	if h == nil { // subsequent builds without adding tracer to avoid duplicate /metrics
		h = server.Default(server.WithHostPorts(getAddr(cfg)))
	}
	for _, m := range common.Middlewares() {
		h.Use(m)
	}
	// project headers middleware
	h.Use(func(c context.Context, ctx *app.RequestContext) {
		ctx.Response.Header.Set("X-AssistFusion-Project", common.ProjectName)
		ctx.Response.Header.Set("X-AssistFusion-Version", common.ProjectVersion)
		ctx.Next(c)
	})
	// domain metrics snapshot under separate path to avoid polluting standard prometheus namespace
	h.GET("/metrics/domain", func(c context.Context, ctx *app.RequestContext) {
		ctx.Response.Header.Set("Content-Type", "text/plain; charset=utf-8")
		ctx.Write([]byte(observability.Snapshot()))
	})
	registerHealthRoutes(h)
	// Decide adapter mode: local (monolith style) or RPC
	if cfg.FeatureRPC {
		ad, err := gateway.NewRPCAdapter(cfg)
		if err != nil {
			log.Printf("failed to init RPC adapter, fallback to local: %v", err)
		} else {
			log.Printf("gateway running in RPC mode")
			registerTicketRoutesRPC(h, ad.Ticket)
			registerKBRoutesRPC(h, ad.KB)
			registerAIRoutesRPC(h, ad.AI)
			return h
		}
	}
	log.Printf("gateway running in LOCAL mode")
	registerTicketRoutes(h, repo)
	registerKBRoutes(h, kbRepo)
	registerAIRoutes(h)
	return h
}

// override registerHealthRoutes to use esInitOK
func registerHealthRoutes(h *server.Hertz) {
	h.GET("/health", func(c context.Context, ctx *app.RequestContext) { ctx.JSON(200, map[string]any{"status": "ok"}) })
	h.GET("/ready", func(c context.Context, ctx *app.RequestContext) {
		// Base readiness object
		// Scenarios:
		// 1. Using memory backend only -> ready
		// 2. ES selected & init succeeded -> ping to confirm live; fallback to degraded if ping fails
		// 3. ES selected & init failed -> degraded memory-fallback
		if esRepoInstance != nil { // ES was intended
			if !esInitOK { // init failed earlier
				ctx.JSON(503, map[string]any{"status": "degraded", "kb": "memory-fallback", "es": "init-failed"})
				return
			}
			// perform ping with tight timeout (inherit request context)
			pingCtx, cancel := context.WithTimeout(c, 400*time.Millisecond)
			defer cancel()
			if err := esRepoInstance.Ping(pingCtx); err != nil {
				ctx.JSON(503, map[string]any{"status": "degraded", "kb": "memory-fallback", "es": "ping-failed", "error": err.Error()})
				return
			}
			ctx.JSON(200, map[string]any{"status": "ready", "backend": "es"})
			return
		}
		// memory only
		ctx.JSON(200, map[string]any{"status": "ready", "backend": "memory"})
	})
}

// --- route registration helpers ---

func registerTicketRoutes(h *server.Hertz, repo common.TicketRepo) {
	h.POST(pathTickets, func(c context.Context, ctx *app.RequestContext) {
		var req struct {
			Title string `json:"title"`
			Desc  string `json:"desc"`
			Note  string `json:"note"`
		}
		if err := ctx.Bind(&req); err != nil {
			common.WriteError(c, ctx, 400, common.ErrCodeBadRequest, badRequestMsg)
			return
		}
		now := time.Now().Unix()
		t := &common.Ticket{
			ID:           uuid.NewString(),
			Title:        req.Title,
			Desc:         req.Desc,
			Status:       "created",
			CreatedAt:    now,
			Cycles:       []common.TicketCycle{{CreatedAt: now, Status: "created"}},
			CurrentCycle: 0,
			Events:       []common.TicketEvent{{Type: "created", At: now, Note: req.Note}},
		}
		repo.Create(c, t)
		observability.TicketCreated.Add(1)
		ctx.JSON(201, t)
	})

	h.GET(pathTickets, func(c context.Context, ctx *app.RequestContext) {
		ts, _ := repo.List(c)
		ctx.JSON(200, ts)
	})

	h.GET(pathTicketID, func(c context.Context, ctx *app.RequestContext) {
		handleGetTicket(c, ctx, repo)
	})

	h.PUT("/v1/tickets/:id/assign", func(c context.Context, ctx *app.RequestContext) {
		handleAssign(c, ctx, repo)
	})

	h.PUT("/v1/tickets/:id/resolve", func(c context.Context, ctx *app.RequestContext) {
		handleResolve(c, ctx, repo)
	})

	h.PUT("/v1/tickets/:id/escalate", func(c context.Context, ctx *app.RequestContext) {
		handleEscalate(c, ctx, repo)
	})

	// reopen a resolved ticket when it was solved incorrectly
	h.PUT("/v1/tickets/:id/reopen", func(c context.Context, ctx *app.RequestContext) {
		handleReopen(c, ctx, repo)
	})

	// list cycles of a ticket (history rounds)
	h.GET("/v1/tickets/:id/cycles", func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, _ := repo.Get(c, id)
		if t == nil {
			common.WriteError(c, ctx, 404, common.ErrCodeNotFound, notFoundMsg)
			return
		}
		ctx.JSON(200, map[string]any{
			"current": t.CurrentCycle,
			"cycles":  t.Cycles,
		})
	})

	// events audit trail
	h.GET("/v1/tickets/:id/events", func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, _ := repo.Get(c, id)
		if t == nil {
			ctx.JSON(404, map[string]string{"error": notFoundMsg})
			return
		}
		ctx.JSON(200, map[string]any{"events": t.Events})
	})
}

// --- RPC route variants (using gateway.TicketAPI etc.) ---
func registerTicketRoutesRPC(h *server.Hertz, api gateway.TicketAPI) {
	h.POST("/v1/tickets", func(c context.Context, ctx *app.RequestContext) {
		var req struct {
			Title string `json:"title"`
			Desc  string `json:"desc"`
			Note  string `json:"note"`
		}
		if err := ctx.Bind(&req); err != nil || req.Title == "" {
			common.WriteError(c, ctx, 400, common.ErrCodeBadRequest, badRequestMsg)
			return
		}
		t, err := api.Create(c, req.Title, req.Desc, req.Note)
		if err != nil {
			writeServiceError(c, ctx, err)
			return
		}
		observability.TicketCreated.Add(1)
		ctx.JSON(201, t)
	})
	h.GET("/v1/tickets", func(c context.Context, ctx *app.RequestContext) {
		ts, err := api.List(c)
		if err != nil {
			writeServiceError(c, ctx, err)
			return
		}
		ctx.JSON(200, ts)
	})
	h.GET("/v1/tickets/:id", func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		t, err := api.Get(c, id)
		if err != nil {
			writeServiceError(c, ctx, err)
			return
		}
		if t == nil {
			common.WriteError(c, ctx, 404, common.ErrCodeNotFound, notFoundMsg)
			return
		}
		ctx.JSON(200, t)
	})
	h.PUT(pathTicketID+"/assign", func(c context.Context, ctx *app.RequestContext) { ticketActionRPC(c, ctx, api.Assign) })
	h.PUT(pathTicketID+"/resolve", func(c context.Context, ctx *app.RequestContext) { ticketActionRPC(c, ctx, api.Resolve) })
	h.PUT(pathTicketID+"/escalate", func(c context.Context, ctx *app.RequestContext) { ticketActionRPC(c, ctx, api.Escalate) })
	h.PUT(pathTicketID+"/reopen", func(c context.Context, ctx *app.RequestContext) { ticketActionRPC(c, ctx, api.Reopen) })
	h.GET(pathTicketCycles, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		cs, err := api.Cycles(c, id)
		if err != nil {
			writeServiceError(c, ctx, err)
			return
		}
		// Derive current index (RPC returns in ticket struct but we only have cycles list here)
		ctx.JSON(200, map[string]any{"cycles": cs})
	})
	h.GET(pathTicketEvents, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		es, err := api.Events(c, id)
		if err != nil {
			writeServiceError(c, ctx, err)
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
		writeServiceError(c, ctx, err)
		return
	}
	ctx.JSON(200, t)
}

// Map RPC ServiceError into our HTTP error schema.
func writeServiceError(c context.Context, ctx *app.RequestContext, err error) {
	if err == nil {
		return
	}
	// naive type assertion on generated ServiceError (kitex_gen/common)
	if se, ok := err.(*kcommon.ServiceError); ok {
		code := 500
		switch se.Code {
		case "bad_request":
			code = 400
		case "not_found":
			code = 404
		case "conflict":
			code = 409
		case "kb_unavailable":
			code = 503
		}
		common.WriteError(c, ctx, code, se.Code, se.Message)
		return
	}
	common.WriteError(c, ctx, 500, common.ErrCodeInternal, internalErrMsg)
}

func handleGetTicket(c context.Context, ctx *app.RequestContext, repo common.TicketRepo) {
	id := string(ctx.Param("id"))
	t, _ := repo.Get(c, id)
	if t == nil {
		common.WriteError(c, ctx, 404, common.ErrCodeNotFound, notFoundMsg)
		return
	}
	ctx.JSON(200, t)
}

func handleAssign(c context.Context, ctx *app.RequestContext, repo common.TicketRepo) {
	id := string(ctx.Param("id"))
	t, _ := repo.Get(c, id)
	if t == nil {
		common.WriteError(c, ctx, 404, common.ErrCodeNotFound, notFoundMsg)
		return
	}
	// optional note
	var req struct {
		Note string `json:"note"`
	}
	if b := ctx.Request.Body(); len(b) > 0 {
		// 容错：忽略解析错误，保持后向兼容
		_ = ctx.Bind(&req)
	}
	now := time.Now().Unix()
	t.AssignedAt = now
	t.Status = "assigned"
	// update cycle
	if t.CurrentCycle >= 0 && t.CurrentCycle < len(t.Cycles) {
		cyc := &t.Cycles[t.CurrentCycle]
		cyc.AssignedAt = now
		cyc.Status = "assigned"
	}
	// append event then persist
	t.Events = append(t.Events, common.TicketEvent{Type: "assigned", At: now, Note: req.Note})
	repo.Update(c, t)
	observability.TicketAssigned.Add(1)
	ctx.JSON(200, t)
}

func handleResolve(c context.Context, ctx *app.RequestContext, repo common.TicketRepo) {
	id := string(ctx.Param("id"))
	t, _ := repo.Get(c, id)
	if t == nil {
		common.WriteError(c, ctx, 404, common.ErrCodeNotFound, notFoundMsg)
		return
	}
	var req struct {
		Note string `json:"note"`
	}
	if b := ctx.Request.Body(); len(b) > 0 {
		_ = ctx.Bind(&req)
	}
	// resolve is a terminal state; clear ongoing escalation timestamp if any
	now := time.Now().Unix()
	t.ResolvedAt = now
	t.EscalatedAt = 0
	t.Status = "resolved"
	if t.CurrentCycle >= 0 && t.CurrentCycle < len(t.Cycles) {
		cyc := &t.Cycles[t.CurrentCycle]
		cyc.ResolvedAt = now
		cyc.EscalatedAt = 0
		cyc.Status = "resolved"
	}
	// append event then persist
	t.Events = append(t.Events, common.TicketEvent{Type: "resolved", At: now, Note: req.Note})
	repo.Update(c, t)
	observability.TicketResolved.Add(1)
	ctx.JSON(200, t)
}

func handleEscalate(c context.Context, ctx *app.RequestContext, repo common.TicketRepo) {
	id := string(ctx.Param("id"))
	t, _ := repo.Get(c, id)
	if t == nil {
		ctx.JSON(404, map[string]string{"error": notFoundMsg})
		return
	}
	// disallow escalate after resolved
	if t.Status == "resolved" {
		common.WriteError(c, ctx, 409, common.ErrCodeConflict, "cannot escalate resolved ticket")
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
	// append event then persist
	t.Events = append(t.Events, common.TicketEvent{Type: "escalated", At: now, Note: req.Note})
	repo.Update(c, t)
	observability.TicketEscalated.Add(1)
	ctx.JSON(200, t)
}

func handleReopen(c context.Context, ctx *app.RequestContext, repo common.TicketRepo) {
	id := string(ctx.Param("id"))
	t, _ := repo.Get(c, id)
	if t == nil {
		common.WriteError(c, ctx, 404, common.ErrCodeNotFound, notFoundMsg)
		return
	}
	if t.Status != "resolved" {
		common.WriteError(c, ctx, 409, common.ErrCodeConflict, "can only reopen resolved ticket")
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
	// new cycle
	t.Cycles = append(t.Cycles, common.TicketCycle{CreatedAt: now, Status: "created"})
	t.CurrentCycle = len(t.Cycles) - 1
	// sync top-level snapshot to current cycle (B keeps snapshot for compatibility)
	t.Status = "created"
	t.AssignedAt = 0
	t.ResolvedAt = 0
	t.EscalatedAt = 0
	// append event then persist
	t.Events = append(t.Events, common.TicketEvent{Type: "reopened", At: now, Note: req.Note})
	repo.Update(c, t)
	observability.TicketReopened.Add(1)
	ctx.JSON(200, t)
}

func registerKBRoutes(h *server.Hertz, kbRepo kb.Repo) {
	h.POST(pathDocs, kbPostDocHandler(kbRepo))
	h.PUT(pathDocID, kbPutDocHandler(kbRepo))
	h.DELETE(pathDocID, kbDeleteDocHandler(kbRepo))
	h.GET(pathSearch, kbSearchHandler(kbRepo))
	h.GET(pathKBInfo, kbInfoHandler(kbRepo))
}

func registerKBRoutesRPC(h *server.Hertz, api gateway.KBAPI) {
	h.POST(pathDocs, kbAddRPCHandler(api))
	h.PUT(pathDocID, kbUpdateRPCHandler(api))
	h.DELETE(pathDocID, kbDeleteRPCHandler(api))
	h.GET(pathSearch, kbSearchRPCHandler(api))
	h.GET(pathKBInfo, kbInfoRPCHandler(api))
}

func kbAddRPCHandler(api gateway.KBAPI) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		var req struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		if err := ctx.Bind(&req); err != nil || req.Title == "" {
			common.WriteError(c, ctx, 400, common.ErrCodeBadRequest, badRequestMsg)
			return
		}
		d, err := api.Add(c, req.Title, req.Content)
		if err != nil {
			writeServiceError(c, ctx, err)
			return
		}
		observability.KBDocCreated.Add(1)
		ctx.JSON(201, map[string]string{"id": d.Id})
	}
}
func kbUpdateRPCHandler(api gateway.KBAPI) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		var req struct {
			Title   *string `json:"title"`
			Content *string `json:"content"`
		}
		if b := ctx.Request.Body(); len(b) > 0 {
			_ = ctx.Bind(&req)
		}
		title, content := "", ""
		if req.Title != nil {
			title = *req.Title
		}
		if req.Content != nil {
			content = *req.Content
		}
		d, err := api.Update(c, id, title, content)
		if err != nil {
			writeServiceError(c, ctx, err)
			return
		}
		observability.KBDocUpdated.Add(1)
		ctx.JSON(200, map[string]any{"id": d.Id})
	}
}
func kbDeleteRPCHandler(api gateway.KBAPI) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		if err := api.Delete(c, id); err != nil {
			writeServiceError(c, ctx, err)
			return
		}
		observability.KBDocDeleted.Add(1)
		ctx.JSON(204, nil)
	}
}
func kbSearchRPCHandler(api gateway.KBAPI) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		q := string(ctx.Query("q"))
		limit := int32(10)
		if v := ctx.Query("limit"); len(v) > 0 {
			if n, err := strconv.Atoi(string(v)); err == nil && n > 0 {
				if n > 50 {
					n = 50
				}
				limit = int32(n)
			}
		}
		items, err := api.Search(c, q, limit)
		if err != nil {
			writeServiceError(c, ctx, err)
			return
		}
		observability.KBSearchRequests.Add(1)
		observability.KBSearchHits.Add(int64(len(items)))
		ctx.JSON(200, map[string]any{"items": items, "total": len(items)})
	}
}
func kbInfoRPCHandler(api gateway.KBAPI) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		info, err := api.Info(c)
		if err != nil {
			writeServiceError(c, ctx, err)
			return
		}
		ctx.JSON(200, info)
	}
}

func kbPostDocHandler(kbRepo kb.Repo) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		var req struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		if err := ctx.Bind(&req); err != nil || req.Title == "" {
			common.WriteError(c, ctx, 400, common.ErrCodeBadRequest, badRequestMsg)
			return
		}
		d := &kb.Doc{ID: uuid.NewString(), Title: req.Title, Content: req.Content}
		if err := kbRepo.Add(c, d); err != nil {
			common.WriteError(c, ctx, 503, common.ErrCodeKBUnavailable, kbUnavailableMsg)
			return
		}
		observability.KBDocCreated.Add(1)
		ctx.JSON(201, map[string]string{"id": d.ID})
	}
}

func kbPutDocHandler(kbRepo kb.Repo) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		if id == "" {
			common.WriteError(c, ctx, 400, common.ErrCodeBadRequest, badRequestMsg)
			return
		}
		req, ok := bindKBDocPartial(ctx)
		if !ok {
			ctx.JSON(400, map[string]string{"error": badRequestMsg})
			return
		}
		d, ok := kbRepo.Get(c, id)
		if !ok {
			d = &kb.Doc{ID: id}
		}
		applyKBDocPartial(d, req)
		if d.Title == "" {
			common.WriteError(c, ctx, 400, common.ErrCodeBadRequest, badRequestMsg)
			return
		}
		if err := kbRepo.Update(c, d); err != nil {
			common.WriteError(c, ctx, 500, common.ErrCodeInternal, "internal error")
			return
		}
		observability.KBDocUpdated.Add(1)
		ctx.JSON(200, map[string]any{"id": d.ID})
	}
}

type kbDocPartial struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}

func bindKBDocPartial(ctx *app.RequestContext) (*kbDocPartial, bool) {
	var req kbDocPartial
	if b := ctx.Request.Body(); len(b) > 0 {
		if err := ctx.Bind(&req); err != nil {
			return nil, false
		}
	}
	return &req, true
}

func applyKBDocPartial(d *kb.Doc, p *kbDocPartial) {
	if p == nil {
		return
	}
	if p.Title != nil {
		d.Title = *p.Title
	}
	if p.Content != nil {
		d.Content = *p.Content
	}
}

func kbDeleteDocHandler(kbRepo kb.Repo) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		if id == "" {
			common.WriteError(c, ctx, 400, common.ErrCodeBadRequest, badRequestMsg)
			return
		}
		if err := kbRepo.Delete(c, id); err != nil {
			common.WriteError(c, ctx, 503, common.ErrCodeKBUnavailable, kbUnavailableMsg)
			return
		}
		observability.KBDocDeleted.Add(1)
		ctx.JSON(204, nil)
	}
}

func kbSearchHandler(kbRepo kb.Repo) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		q := string(ctx.Query("q"))
		// optional limit query param
		limit := 10
		if v := ctx.Query("limit"); len(v) > 0 {
			if n, err := strconv.Atoi(string(v)); err == nil && n > 0 {
				if n > 50 {
					n = 50
				}
				limit = n
			}
		}
		items, total, err := kbRepo.Search(c, q, limit)
		if err != nil {
			common.WriteError(c, ctx, 503, common.ErrCodeKBUnavailable, kbUnavailableMsg)
			return
		}
		observability.KBSearchRequests.Add(1)
		observability.KBSearchHits.Add(int64(len(items)))
		ctx.JSON(200, map[string]any{"items": items, "total": total})
	}
}

// optional info handler: if repo supports Info(), expose minimal diagnostics
type kbInfo interface {
	Info(ctx context.Context) (map[string]any, error)
}

func kbInfoHandler(kbRepo kb.Repo) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		if r, ok := kbRepo.(kbInfo); ok {
			info, err := r.Info(c)
			if err != nil {
				common.WriteError(c, ctx, 500, common.ErrCodeInternal, "internal error")
				return
			}
			ctx.JSON(200, info)
			return
		}
		ctx.JSON(200, map[string]any{"backend": "memory"})
	}
}

func registerAIRoutes(h *server.Hertz) {
	h.POST(pathEmbeddings, func(c context.Context, ctx *app.RequestContext) {
		var req struct {
			Texts []string `json:"texts"`
			Dim   int      `json:"dim"`
		}
		if err := ctx.Bind(&req); err != nil || len(req.Texts) == 0 {
			common.WriteError(c, ctx, 400, common.ErrCodeBadRequest, badRequestMsg)
			return
		}
		vecs := ai.MockEmbeddings(req.Texts, req.Dim)
		observability.AIEmbeddingCalls.Add(1)
		ctx.JSON(200, map[string]any{"vectors": vecs, "dim": len(vecs[0])})
	})
}

func registerAIRoutesRPC(h *server.Hertz, api gateway.AIAPI) {
	h.POST(pathEmbeddings, func(c context.Context, ctx *app.RequestContext) {
		var req struct {
			Texts []string `json:"texts"`
			Dim   int32    `json:"dim"`
		}
		if err := ctx.Bind(&req); err != nil || len(req.Texts) == 0 {
			common.WriteError(c, ctx, 400, common.ErrCodeBadRequest, badRequestMsg)
			return
		}
		r, err := api.Embeddings(c, req.Texts, req.Dim)
		if err != nil {
			writeServiceError(c, ctx, err)
			return
		}
		observability.AIEmbeddingCalls.Add(1)
		ctx.JSON(200, map[string]any{"vectors": r.Vectors, "dim": r.Dim})
	})
}

// test helper (not exposed in production builds): startTestServer returns server and bound address
func startTestServer(t interface {
	Fatalf(format string, args ...any)
}) (*server.Hertz, string) {
	cfg := common.LoadConfig()
	// choose an ephemeral port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	cfg.HTTPAddr = addr
	// force ES backend if env demands; else memory
	h := BuildServer(cfg)
	go h.Spin()
	return h, addr
}
