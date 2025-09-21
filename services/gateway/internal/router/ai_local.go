package router

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"github.com/gogogo1024/assist-fusion/internal/ai"
	"github.com/gogogo1024/assist-fusion/internal/common"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	gwerrors "github.com/gogogo1024/assist-fusion/services/gateway/internal/errors"
)

// RegisterAILocal registers local (mock) AI / embeddings endpoints.
func RegisterAILocal(h *server.Hertz) {
	h.POST(PathEmbeddings, func(c context.Context, ctx *app.RequestContext) {
		var req struct {
			Texts []string `json:"texts"`
			Dim   int      `json:"dim"`
		}
		if err := ctx.Bind(&req); err != nil || len(req.Texts) == 0 {
			gwerrors.HTTPError(ctx, http.StatusBadRequest, common.ErrCodeBadRequest, gwerrors.MsgBadRequest)
			return
		}
		if req.Dim == 0 {
			req.Dim = 128
		}
		if req.Dim < 4 || req.Dim > 4096 {
			gwerrors.HTTPError(ctx, http.StatusBadRequest, common.ErrCodeBadRequest, "invalid dim")
			return
		}
		vecs := ai.MockEmbeddings(req.Texts, req.Dim)
		observability.AIEmbeddingCalls.Add(1)
		ctx.JSON(http.StatusOK, map[string]any{"vectors": vecs, "dim": len(vecs[0])})
	})
}
