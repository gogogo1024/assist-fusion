package router

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"github.com/gogogo1024/assist-fusion/internal/common"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	aicli "github.com/gogogo1024/assist-fusion/kitex_gen/ai/aiservice"
	commonidl "github.com/gogogo1024/assist-fusion/kitex_gen/common"
	gwerrors "github.com/gogogo1024/assist-fusion/services/gateway/internal/errors"
)

// DepsAI minimal interface for AI RPC registration.
type DepsAI interface{ AIClient() aicli.Client }

// RegisterAIRPC registers AI embeddings endpoints backed by RPC service.
func RegisterAIRPC(h *server.Hertz, deps DepsAI) {
	cli := deps.AIClient()
	h.POST(PathEmbeddings, func(c context.Context, ctx *app.RequestContext) {
		var req struct {
			Texts []string `json:"texts"`
			Dim   int32    `json:"dim"`
		}
		if err := ctx.Bind(&req); err != nil || len(req.Texts) == 0 {
			gwerrors.HTTPError(ctx, http.StatusBadRequest, common.ErrCodeBadRequest, gwerrors.MsgBadRequest)
			return
		}
		// default dim
		if req.Dim == 0 {
			req.Dim = 128
		}
		r, err := cli.Embeddings(c, &commonidl.EmbeddingRequest{Texts: req.Texts, Dim: req.Dim})
		if err != nil || r == nil {
			gwerrors.HTTPError(ctx, http.StatusInternalServerError, common.ErrCodeInternal, gwerrors.MsgInternal)
			return
		}
		observability.AIEmbeddingCalls.Add(1)
		ctx.JSON(http.StatusOK, map[string]any{"vectors": r.Vectors, "dim": r.Dim})
	})
}
