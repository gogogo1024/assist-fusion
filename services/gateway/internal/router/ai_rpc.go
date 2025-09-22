package router

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"github.com/gogogo1024/assist-fusion/internal/ai/chain"
	"github.com/gogogo1024/assist-fusion/internal/common"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	aidl "github.com/gogogo1024/assist-fusion/kitex_gen/ai"
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

	// Streaming Chat SSE endpoint (best-effort; falls back to non-stream if provider doesn't support)
	h.POST("/api/ai/chat/stream", func(c context.Context, ctx *app.RequestContext) {
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			MaxTokens *int `json:"max_tokens"`
		}
		if err := ctx.Bind(&req); err != nil || len(req.Messages) == 0 {
			gwerrors.HTTPError(ctx, http.StatusBadRequest, common.ErrCodeBadRequest, gwerrors.MsgBadRequest)
			return
		}
		// We need direct streaming only if AI provider is openai; otherwise emulate with one-shot.
		provider := chain.DetectProvider()
		if provider != chain.ProviderOpenAI {
			// fallback single response
			// Convert to RPC Chat
			cr := make([]*aidl.ChatMessage, 0, len(req.Messages))
			for _, m := range req.Messages {
				cr = append(cr, &aidl.ChatMessage{Role: m.Role, Content: m.Content})
			}
			r, err := cli.Chat(c, &aidl.ChatRequest{Messages: cr})
			if err != nil || r == nil || r.Message == nil {
				gwerrors.HTTPError(ctx, http.StatusInternalServerError, common.ErrCodeInternal, gwerrors.MsgInternal)
				return
			}
			ctx.JSON(http.StatusOK, map[string]string{"role": r.Message.Role, "content": r.Message.Content, "provider": provider})
			return
		}
		// Provider is openai -> we re-run streaming at chain layer (bypassing RPC Chat which is non-stream). For simplicity, reconstruct chain embedding chat.
		chatChain := chain.NewChatChain(provider)
		msgs := make([]chain.ChatMessage, 0, len(req.Messages))
		for _, m := range req.Messages {
			msgs = append(msgs, chain.ChatMessage{Role: m.Role, Content: m.Content})
		}
		ctx.SetContentType("text/event-stream")
		ctx.SetStatusCode(http.StatusOK)
		ctx.Response.Header.Add("Cache-Control", "no-cache")
		ctx.Response.Header.Add("Connection", "keep-alive")
		flusher := ctx
		var full strings.Builder
		writeEvent := func(data string) {
			evt := "data:" + data + "\n\n"
			ctx.Response.AppendBodyString(evt)
			flusher.Flush()
		}
		maxTokens := 0
		if req.MaxTokens != nil {
			maxTokens = *req.MaxTokens
		}
		done := make(chan struct{})
		go func() {
			_, _ = chatChain.ChatStream(c, msgs, maxTokens, func(delta string) bool {
				if delta != "" {
					full.WriteString(delta)
					b, _ := json.Marshal(map[string]string{"delta": delta})
					writeEvent(string(b))
				}
				return true
			})
			b, _ := json.Marshal(map[string]any{"done": true, "content": full.String(), "provider": provider})
			writeEvent(string(b))
			ctx.Response.AppendBodyString("data:[DONE]\n\n")
			flusher.Flush()
			close(done)
		}()
		// simple timeout (client may disconnect earlier)
		select {
		case <-c.Done():
		case <-done:
		case <-time.After(60 * time.Second):
		}
	})
}
