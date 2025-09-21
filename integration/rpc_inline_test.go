package integration

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/server"
	"github.com/gogogo1024/assist-fusion/internal/common"
	kbstore "github.com/gogogo1024/assist-fusion/internal/kb"
	"github.com/gogogo1024/assist-fusion/kitex_gen/ai/aiservice"
	kcommon "github.com/gogogo1024/assist-fusion/kitex_gen/common"
	kbidl "github.com/gogogo1024/assist-fusion/kitex_gen/kb"
	"github.com/gogogo1024/assist-fusion/kitex_gen/kb/kbservice"
	"github.com/gogogo1024/assist-fusion/kitex_gen/ticket"
	"github.com/gogogo1024/assist-fusion/kitex_gen/ticket/ticketservice"
	aiimpl "github.com/gogogo1024/assist-fusion/rpc/ai/impl"
	kbimpl "github.com/gogogo1024/assist-fusion/rpc/kb/impl"
	ticketimpl "github.com/gogogo1024/assist-fusion/rpc/ticket/impl"
)

// helper to launch a kitex server on :0 and return address and a stop func.
func launch(t *testing.T, newServer func() server.Server) string {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	svr := newServer()
	go func() { _ = svr.Run() }()
	return ln.Addr().String() // kitex listens on provided addr
}

func freePort(t *testing.T) string {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("freePort listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return fmt.Sprintf(":%d", port)
}
func mustPort(addr string) int {
	var p int
	if _, err := fmt.Sscanf(addr, ":%d", &p); err != nil {
		panic(err)
	}
	return p
}

func TestInlineRPCBasic(t *testing.T) {
	const loopback = "127.0.0.1"
	// ticket
	ticketRepo := common.NewMemoryTicketRepo()
	ticketHandler := ticketimpl.NewTicketService(ticketRepo)
	tPortAddr := freePort(t)
	tSrv := ticketservice.NewServer(ticketHandler, server.WithServiceAddr(&net.TCPAddr{Port: mustPort(tPortAddr)}))
	go func() { _ = tSrv.Run() }()
	ticketAddr := loopback + tPortAddr

	// kb
	kbRepo := kbstore.NewMemoryRepo()
	kbHandler := kbimpl.NewKBService(kbRepo)
	kPortAddr := freePort(t)
	kbSrv := kbservice.NewServer(kbHandler, server.WithServiceAddr(&net.TCPAddr{Port: mustPort(kPortAddr)}))
	go func() { _ = kbSrv.Run() }()
	kbAddr := loopback + kPortAddr

	// ai
	aiHandler := aiimpl.NewAIService()
	aPortAddr := freePort(t)
	aiSrv := aiservice.NewServer(aiHandler, server.WithServiceAddr(&net.TCPAddr{Port: mustPort(aPortAddr)}))
	go func() { _ = aiSrv.Run() }()
	aiAddr := loopback + aPortAddr

	// small readiness wait
	time.Sleep(300 * time.Millisecond)

	tCli, err := ticketservice.NewClient("ticket-rpc", client.WithHostPorts(ticketAddr))
	if err != nil {
		t.Fatalf("ticket client: %v", err)
	}
	kbCli, err := kbservice.NewClient("kb-rpc", client.WithHostPorts(kbAddr))
	if err != nil {
		t.Fatalf("kb client: %v", err)
	}
	aiCli, err := aiservice.NewClient("ai-rpc", client.WithHostPorts(aiAddr))
	if err != nil {
		t.Fatalf("ai client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ticket create
	tr, err := tCli.CreateTicket(ctx, &ticket.CreateTicketRequest{Title: "t", Desc: "d"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if tr.GetTicket().GetId() == "" {
		t.Fatalf("empty ticket id")
	}

	// KB add & search
	doc, err := kbCli.AddDoc(ctx, &kbidl.AddDocRequest{Title: "doc", Content: "abc"})
	if err != nil {
		t.Fatalf("AddDoc: %v", err)
	}
	if doc.GetId() == "" {
		t.Fatalf("empty doc id")
	}
	_, err = kbCli.Search(ctx, &kbidl.SearchRequest{Query: "doc"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	// Embeddings
	eresp, err := aiCli.Embeddings(ctx, &kcommon.EmbeddingRequest{Texts: []string{"x"}, Dim: 4})
	if err != nil {
		t.Fatalf("Embeddings: %v", err)
	}
	if eresp.GetDim() != 4 {
		t.Fatalf("expected dim=4 got %d", eresp.GetDim())
	}
}
