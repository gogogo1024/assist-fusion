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
	registerKBDocCRUD(h, cli)
	registerKBSearch(h, cli)
	registerKBInfo(h, cli)
}

func registerKBDocCRUD(h *server.Hertz, cli kbcli.Client) {
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
		if req.Title == nil || *req.Title == "" {
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
}

func registerKBSearch(h *server.Hertz, cli kbcli.Client) {
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
		var offset *int32
		if v := ctx.Query("offset"); len(v) > 0 {
			if n, err := strconv.Atoi(string(v)); err == nil && n >= 0 {
				val := int32(n)
				offset = &val
			}
		}
		req := &kb.SearchRequest{Query: q}
		if limit > 0 {
			req.Limit = &limit
		}
		if offset != nil {
			req.Offset = offset
		}
		resp, err := cli.Search(c, req)
		if err != nil || resp == nil {
			gwerrors.HTTPError(ctx, http.StatusServiceUnavailable, common.ErrCodeKBUnavailable, gwerrors.MsgKBUnavailable)
			return
		}
		observability.KBSearchRequests.Add(1)
		observability.KBSearchHits.Add(int64(resp.Returned))
		totalVal := resp.Returned
		if resp.Total != nil && *resp.Total >= resp.Returned { // prefer authoritative total when present
			totalVal = *resp.Total
		}
		body := map[string]any{"items": resp.Items, "returned": resp.Returned, "total": totalVal}
		if resp.NextOffset != nil {
			body["next_offset"] = *resp.NextOffset
		}
		ctx.JSON(http.StatusOK, body)
	})
}

func registerKBInfo(h *server.Hertz, cli kbcli.Client) {
	h.GET(PathKBInfo, func(c context.Context, ctx *app.RequestContext) {
		resp, err := cli.Info(c)
		if err != nil || resp == nil {
			gwerrors.HTTPError(ctx, http.StatusServiceUnavailable, common.ErrCodeKBUnavailable, gwerrors.MsgKBUnavailable)
			return
		}
		body := map[string]any{}
		for k, v := range resp.Stats {
			body[k] = v
		}
		ctx.JSON(http.StatusOK, body)
	})
}
