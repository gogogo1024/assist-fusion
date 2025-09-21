package common

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	kerrors "github.com/cloudwego/kitex/pkg/kerrors"
)

// ErrorResponse harmonized HTTP error schema.
type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// RequestIDKey for context retrieval (kept for compatibility with previous middleware design).
const RequestIDKey = "request_id"

// KitexErrorKey stores the constructed kitex biz error in context for further logging.
const KitexErrorKey = "kitex_error"

// MapErrorCodeToHTTP maps domain error codes to HTTP status.
func MapErrorCodeToHTTP(code string) int32 {
	switch code {
	case ErrCodeBadRequest:
		return 400
	case ErrCodeNotFound:
		return 404
	case ErrCodeConflict:
		return 409
	case ErrCodeKBUnavailable:
		return 503
	case ErrCodeInternal:
		return 500
	default:
		return 500
	}
}

// WriteError converts internal error code + message to HTTP JSON response and attaches a Kitex biz error.
func WriteError(c context.Context, ctx *app.RequestContext, status int, code, msg string) {
	// optional request id
	rid := ""
	if v, ok := ctx.Get(RequestIDKey); ok {
		switch vv := v.(type) {
		case string:
			rid = vv
		case []byte:
			rid = string(vv)
		}
	}
	codeInt := MapErrorCodeToHTTP(code)
	if status == 0 {
		status = int(codeInt)
	}
	err := kerrors.NewBizStatusError(codeInt, msg)
	ctx.SetStatusCode(status)
	ctx.JSON(status, ErrorResponse{Code: code, Message: msg, RequestID: rid})
	ctx.Set(KitexErrorKey, err)
}
