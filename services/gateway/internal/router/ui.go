package router

import (
	"context"
	"io/fs"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"github.com/gogogo1024/assist-fusion/internal/common"
	gwerrors "github.com/gogogo1024/assist-fusion/services/gateway/internal/errors"
)

// UIFSProvider minimal interface for providing the embedded UI filesystem.
type UIFSProvider interface{ UI() fs.FS }

// RegisterUI registers SPA/static UI routes under /ui.
func RegisterUI(h *server.Hertz, prov UIFSProvider) {
	contentFS := prov.UI()
	fileHandler := func(c context.Context, ctx *app.RequestContext) {
		p := string(ctx.Request.URI().Path())
		p = strings.TrimPrefix(p, "/ui")
		if p == "" || p == "/" {
			p = "/index.html"
		}
		if strings.Contains(p, "..") {
			gwerrors.HTTPError(ctx, http.StatusBadRequest, common.ErrCodeBadRequest, gwerrors.MsgBadRequest)
			return
		}
		b, err := fs.ReadFile(contentFS, strings.TrimPrefix(p, "/"))
		if err != nil {
			// fallback to SPA index
			b, err = fs.ReadFile(contentFS, "index.html")
			if err != nil {
				gwerrors.HTTPError(ctx, http.StatusNotFound, common.ErrCodeNotFound, gwerrors.MsgNotFound)
				return
			}
			ctx.Response.Header.Set("Content-Type", "text/html; charset=utf-8")
			ctx.Write(b)
			return
		}
		ctx.Response.Header.Set("Content-Type", guessMime(p))
		if strings.HasSuffix(p, ".js") || strings.HasSuffix(p, ".css") || strings.HasSuffix(p, ".svg") {
			ctx.Response.Header.Set("Cache-Control", "public, max-age=86400")
		}
		ctx.Write(b)
	}
	h.GET("/ui", fileHandler)
	h.GET("/ui/*filepath", fileHandler)
}

// guessMime duplicated minimal helper (could be shared later) – kept private here.
func guessMime(p string) string {
	lp := strings.ToLower(p)
	switch {
	case strings.HasSuffix(lp, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(lp, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(lp, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(lp, ".json"):
		return "application/json; charset=utf-8"
	case strings.HasSuffix(lp, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(lp, ".png"):
		return "image/png"
	case strings.HasSuffix(lp, ".jpg") || strings.HasSuffix(lp, ".jpeg"):
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}
