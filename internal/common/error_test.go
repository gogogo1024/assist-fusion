package common

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

func TestWriteErrorIncludesRequestID(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	_ = ln.Close()
	h := server.New(server.WithHostPorts(addr))
	h.GET("/err", func(c context.Context, ctx *app.RequestContext) {
		ctx.Set(RequestIDKey, "abc-123")
		WriteError(c, ctx, 400, ErrCodeBadRequest, "bad")
	})
	go h.Spin()
	time.Sleep(50 * time.Millisecond)
	defer h.Shutdown(context.Background())
	resp, err := http.Get("http://" + addr + "/err")
	if err != nil {
		t.Fatalf("req: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var er ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if er.Code != ErrCodeBadRequest || er.Message != "bad" || er.RequestID != "abc-123" {
		t.Fatalf("bad body %#v", er)
	}
}

func TestRecoveryMiddlewareProducesError(t *testing.T) {
	Logger = nil
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	_ = ln.Close()
	h := server.New(server.WithHostPorts(addr))
	// insert our middlewares manually
	for _, m := range Middlewares() {
		h.Use(m)
	}
	h.GET("/panic", func(c context.Context, ctx *app.RequestContext) { panic("boom") })
	go h.Spin()
	time.Sleep(50 * time.Millisecond)
	defer h.Shutdown(context.Background())
	req, _ := http.NewRequest(http.MethodGet, "http://"+addr+"/panic", nil)
	req.Header.Set("X-Request-ID", "rid-1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("req: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var er ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if er.Code != ErrCodeInternal || er.RequestID != "rid-1" {
		t.Fatalf("unexpected body %#v", er)
	}
}
