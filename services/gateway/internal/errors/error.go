package errors

import (
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	kerrors "github.com/cloudwego/kitex/pkg/kerrors"

	"github.com/gogogo1024/assist-fusion/internal/common"
)

// Standard short messages reused across handlers (moved from main.go)
const (
	MsgNotFound      = "not found"
	MsgBadRequest    = "bad request"
	MsgKBUnavailable = "kb backend unavailable"
	MsgInternal      = "internal error"
)

// HTTPError writes a JSON error response with unified schema and sets a BizStatusError for tracing/logging.
func HTTPError(ctx *app.RequestContext, status int, code, msg string) {
	if status == 0 {
		status = http.StatusInternalServerError
	}
	ctx.Set("biz_error", kerrors.NewBizStatusError(int32(status), msg))
	ctx.SetStatusCode(status)
	ctx.JSON(status, map[string]any{"code": code, "message": msg})
}

// MapServiceError maps a Kitex generated ServiceError (common.ServiceError) to HTTP response using our schema.
// Fallback to 500/internal when type assertion fails.
func MapServiceError(ctx *app.RequestContext, err error) bool {
	if err == nil {
		return false
	}
	if se, ok := err.(interface {
		GetCode() string
		GetMessage() string
	}); ok {
		codeStr := se.GetCode()
		status := http.StatusInternalServerError
		switch codeStr {
		case "bad_request":
			status = http.StatusBadRequest
		case "not_found":
			status = http.StatusNotFound
		case "conflict":
			status = http.StatusConflict
		case "kb_unavailable":
			status = http.StatusServiceUnavailable
		}
		HTTPError(ctx, status, codeStr, se.GetMessage())
		return true
	}
	HTTPError(ctx, http.StatusInternalServerError, common.ErrCodeInternal, MsgInternal)
	return true
}
