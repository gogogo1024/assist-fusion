package main

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/server"
	"github.com/gogogo1024/assist-fusion/internal/common"
	rpcClients "github.com/gogogo1024/assist-fusion/internal/gateway/rpc"
	kbmem "github.com/gogogo1024/assist-fusion/internal/kb"
	aiidl "github.com/gogogo1024/assist-fusion/kitex_gen/ai"
	"github.com/gogogo1024/assist-fusion/kitex_gen/ai/aiservice"
	kcommon "github.com/gogogo1024/assist-fusion/kitex_gen/common"
	kbidl "github.com/gogogo1024/assist-fusion/kitex_gen/kb"
	"github.com/gogogo1024/assist-fusion/kitex_gen/kb/kbservice"
	ticketidl "github.com/gogogo1024/assist-fusion/kitex_gen/ticket"
	"github.com/gogogo1024/assist-fusion/kitex_gen/ticket/ticketservice"
	aiimpl "github.com/gogogo1024/assist-fusion/rpc/ai/impl"
	kbimpl "github.com/gogogo1024/assist-fusion/rpc/kb/impl"
	ticketimpl "github.com/gogogo1024/assist-fusion/rpc/ticket/impl"
)

// startKitexTestServer launches a minimal kitex server on an ephemeral port and returns addr + stop.
func startKitexTestServer(t *testing.T, service string, handler any) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen %s: %v", service, err)
	}
	portAddr := ln.Addr().String()
	_ = ln.Close()
	tcpAddr, _ := net.ResolveTCPAddr("tcp", portAddr)
	var svr server.Server
	switch service {
	case "ticket":
		svr = ticketservice.NewServer(handler.(ticketidl.TicketService), server.WithServiceAddr(tcpAddr))
	case "kb":
		svr = kbservice.NewServer(handler.(kbidl.KBService), server.WithServiceAddr(tcpAddr))
	case "ai":
		svr = aiservice.NewServer(handler.(aiidl.AIService), server.WithServiceAddr(tcpAddr))
	default:
		t.Fatalf("unknown service %s", service)
	}
	go func() {
		if err := svr.Run(); err != nil {
			klog.Errorf("%s server stopped: %v", service, err)
		}
	}()
	// small readiness delay
	time.Sleep(120 * time.Millisecond)
	return portAddr, func() { svr.Stop() }
}

// startAllRPC starts ticket, kb, ai services and initializes gateway RPC clients.
// startAllRPC spawns in-process rpc services and wires gateway rpc clients via direct host:ports.
func startAllRPC(t *testing.T) (addrs map[string]string, stops []func()) {
	t.Helper()
	addrs = map[string]string{}
	ticketRepo := common.NewMemoryTicketRepo()
	tAddr, stopT := startKitexTestServer(t, "ticket", ticketimpl.NewTicketService(ticketRepo))
	addrs["ticket"] = tAddr
	stops = append(stops, stopT)
	kbRepo := kbmem.NewMemoryRepo()
	kAddr, stopK := startKitexTestServer(t, "kb", kbimpl.NewKBService(kbRepo))
	addrs["kb"] = kAddr
	stops = append(stops, stopK)
	aAddr, stopA := startKitexTestServer(t, "ai", aiimpl.NewAIService())
	addrs["ai"] = aAddr
	stops = append(stops, stopA)

	// Disable consul for tests and init clients with direct host ports.
	os.Setenv("DISABLE_CONSUL", "1")
	cfg := &common.Config{
		TicketRPCAddr: addrs["ticket"],
		KBRPCAddr:     addrs["kb"],
		AIRPCAddr:     addrs["ai"],
	}
	if err := rpcClients.Init(cfg); err != nil {
		t.Fatalf("init rpc clients: %v", err)
	}
	// sanity probe
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := rpcClients.AIClient.Embeddings(ctx, &kcommon.EmbeddingRequest{Texts: []string{"ping"}, Dim: 2}); err != nil {
		t.Fatalf("ai embeddings probe failed: %v", err)
	}
	return addrs, stops
}
