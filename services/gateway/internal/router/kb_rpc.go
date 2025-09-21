package router

import (
	"context"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"github.com/gogogo1024/assist-fusion/internal/common"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	kb "github.com/gogogo1024/assist-fusion/kitex_gen/kb"
	kbcli "github.com/gogogo1024/assist-fusion/kitex_gen/kb/kbservice"
	gwerrors "github.com/gogogo1024/assist-fusion/services/gateway/internal/errors"
)

// DepsKB defines what we need for RPC KB registration (subset of full Deps later)
type DepsKB interface{ KBClient() kbcli.Client }

func RegisterKBRPC(h *server.Hertz, deps DepsKB) {
	cli := deps.KBClient()
	// create doc
	h.POST(PathDocs, func(c context.Context, ctx *app.RequestContext) {
		var req kb.AddDocRequest
		if err := ctx.Bind(&req); err != nil || req.Title == "" {
			gwerrors.HTTPError(ctx, http.StatusBadRequest, common.ErrCodeBadRequest, gwerrors.MsgBadRequest)
			return
		}
		resp, err := cli.AddDoc(c, &req)
		if err != nil || resp == nil || resp.Id == "" {
			gwerrors.HTTPError(ctx, http.StatusServiceUnavailable, common.ErrCodeKBUnavailable, gwerrors.MsgKBUnavailable)
			return
		}
		observability.KBDocCreated.Add(1)
		ctx.JSON(http.StatusCreated, map[string]any{"id": resp.Id})
	})

	// update (partial)
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
		req := &kb.UpdateDocRequest{Id: id}
		if patch.Title != nil {
			req.Title = patch.Title
		}
		if patch.Content != nil {
			req.Content = patch.Content
		}
		if req.Title == nil || *req.Title == "" { // title required after patch
			gwerrors.HTTPError(ctx, http.StatusBadRequest, common.ErrCodeBadRequest, gwerrors.MsgBadRequest)
			return
		}
		if _, err := cli.UpdateDoc(c, req); err != nil {
			gwerrors.HTTPError(ctx, http.StatusInternalServerError, common.ErrCodeInternal, gwerrors.MsgInternal)
			return
		}
		observability.KBDocUpdated.Add(1)
		ctx.JSON(http.StatusOK, map[string]any{"id": id})
	})

	// delete
	h.DELETE(PathDocID, func(c context.Context, ctx *app.RequestContext) {
		id := string(ctx.Param("id"))
		if id == "" {
			gwerrors.HTTPError(ctx, http.StatusBadRequest, common.ErrCodeBadRequest, gwerrors.MsgBadRequest)
			return
		}
		if _, err := cli.DeleteDoc(c, &kb.DeleteDocRequest{Id: id}); err != nil {
			gwerrors.HTTPError(ctx, http.StatusServiceUnavailable, common.ErrCodeKBUnavailable, gwerrors.MsgKBUnavailable)
			return
		}
		observability.KBDocDeleted.Add(1)
		ctx.JSON(http.StatusNoContent, nil)
	})

	// search
	h.GET(PathSearch, func(c context.Context, ctx *app.RequestContext) {
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
		req := &kb.SearchRequest{Query: q}
		if limit > 0 {
			req.Limit = &limit
		}
		resp, err := cli.Search(c, req)
		if err != nil || resp == nil {
			gwerrors.HTTPError(ctx, http.StatusServiceUnavailable, common.ErrCodeKBUnavailable, gwerrors.MsgKBUnavailable)
			return
		}
		observability.KBSearchRequests.Add(1)
		observability.KBSearchHits.Add(int64(resp.Returned))
		ctx.JSON(http.StatusOK, map[string]any{"items": resp.Items, "returned": resp.Returned})
	})

	// info
	h.GET(PathKBInfo, func(c context.Context, ctx *app.RequestContext) {
		resp, err := cli.Info(c)
		if err != nil || resp == nil {
			gwerrors.HTTPError(ctx, http.StatusServiceUnavailable, common.ErrCodeKBUnavailable, gwerrors.MsgKBUnavailable)
			return
		}
		ctx.JSON(http.StatusOK, map[string]any{"stats": resp.Stats})
	})
}
