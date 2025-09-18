package common

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	ErrCodeBadRequest    = "bad_request"
	ErrCodeNotFound      = "not_found"
	ErrCodeConflict      = "conflict"
	ErrCodeKBUnavailable = "kb_unavailable"
	ErrCodeInternal      = "internal_error"
)

type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// RequestIDKey exported for reuse in tests and middleware.
const RequestIDKey = "request_id"

func WriteError(c context.Context, ctx *app.RequestContext, status int, code, msg string) {
	rid := ""
	if v, ok := ctx.Get(RequestIDKey); ok {
		if s, ok2 := v.(string); ok2 {
			rid = s
		}
	}
	ctx.JSON(status, ErrorResponse{Code: code, Message: msg, RequestID: rid})
}
