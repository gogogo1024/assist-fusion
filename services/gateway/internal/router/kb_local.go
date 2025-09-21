package router

import (
	"context"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/google/uuid"

	"github.com/gogogo1024/assist-fusion/internal/common"
	"github.com/gogogo1024/assist-fusion/internal/kb"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	gwerrors "github.com/gogogo1024/assist-fusion/services/gateway/internal/errors"
)

func RegisterKBLocal(h *server.Hertz, repo kb.Repo) {
	// create
	h.POST(PathDocs, func(c context.Context, ctx *app.RequestContext) {
		var req struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		if err := ctx.Bind(&req); err != nil || req.Title == "" {
			gwerrors.HTTPError(ctx, http.StatusBadRequest, common.ErrCodeBadRequest, gwerrors.MsgBadRequest)
			return
		}
		d := &kb.Doc{ID: uuid.NewString(), Title: req.Title, Content: req.Content}
		if err := repo.Add(c, d); err != nil {
			gwerrors.HTTPError(ctx, http.StatusServiceUnavailable, common.ErrCodeKBUnavailable, gwerrors.MsgKBUnavailable)
			return
		}
		observability.KBDocCreated.Add(1)
		ctx.JSON(http.StatusCreated, map[string]string{"id": d.ID})
	})

	// update (partial upsert) – if doc missing we treat as create/replace
	h.PUT(PathDocID, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		if id == "" {
			gwerrors.HTTPError(ctx, http.StatusBadRequest, common.ErrCodeBadRequest, gwerrors.MsgBadRequest)
			return
		}
		var patch struct {
			Title   *string `json:"title"`
			Content *string `json:"content"`
		}
		if b := ctx.Request.Body(); len(b) > 0 {
			if err := ctx.Bind(&patch); err != nil {
				gwerrors.HTTPError(ctx, http.StatusBadRequest, common.ErrCodeBadRequest, gwerrors.MsgBadRequest)
				return
			}
		}
		d, ok := repo.Get(c, id)
		if !ok {
			d = &kb.Doc{ID: id}
		}
		if patch.Title != nil {
			d.Title = *patch.Title
		}
		if patch.Content != nil {
			d.Content = *patch.Content
		}
		if d.Title == "" { // Title required
			gwerrors.HTTPError(ctx, http.StatusBadRequest, common.ErrCodeBadRequest, gwerrors.MsgBadRequest)
			return
		}
		if err := repo.Update(c, d); err != nil {
			gwerrors.HTTPError(ctx, http.StatusInternalServerError, common.ErrCodeInternal, gwerrors.MsgInternal)
			return
		}
		observability.KBDocUpdated.Add(1)
		ctx.JSON(http.StatusOK, map[string]any{"id": d.ID})
	})

	// delete
	h.DELETE(PathDocID, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		if id == "" {
			gwerrors.HTTPError(ctx, http.StatusBadRequest, common.ErrCodeBadRequest, gwerrors.MsgBadRequest)
			return
		}
		if err := repo.Delete(c, id); err != nil {
			gwerrors.HTTPError(ctx, http.StatusServiceUnavailable, common.ErrCodeKBUnavailable, gwerrors.MsgKBUnavailable)
			return
		}
		observability.KBDocDeleted.Add(1)
		ctx.JSON(http.StatusNoContent, nil)
	})

	// search
	h.GET(PathSearch, func(c context.Context, ctx *app.RequestContext) {
		q := string(ctx.Query("q"))
		limit := 10
		if v := ctx.Query("limit"); len(v) > 0 {
			if n, err := strconv.Atoi(string(v)); err == nil && n > 0 {
				if n > 50 { // upper safety bound
					n = 50
				}
				limit = n
			}
		}
		items, total, err := repo.Search(c, q, limit)
		if err != nil {
			gwerrors.HTTPError(ctx, http.StatusServiceUnavailable, common.ErrCodeKBUnavailable, gwerrors.MsgKBUnavailable)
			return
		}
		observability.KBSearchRequests.Add(1)
		observability.KBSearchHits.Add(int64(len(items)))
		ctx.JSON(http.StatusOK, map[string]any{"items": items, "total": total})
	})

	// info
	h.GET(PathKBInfo, func(c context.Context, ctx *app.RequestContext) {
		if r, ok := repo.(interface {
			Info(ctx context.Context) (map[string]any, error)
		}); ok {
			info, err := r.Info(c)
			if err != nil {
				gwerrors.HTTPError(ctx, http.StatusInternalServerError, common.ErrCodeInternal, gwerrors.MsgInternal)
				return
			}
			ctx.JSON(http.StatusOK, info)
			return
		}
		ctx.JSON(http.StatusOK, map[string]any{"backend": "memory"})
	})
}
