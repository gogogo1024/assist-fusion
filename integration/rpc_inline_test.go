package integration

import (
	"context"
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

// NOTE: avoid race-prone freePort pattern by binding listeners first.

func TestInlineRPCBasic(t *testing.T) {
	const anyLoop = "127.0.0.1:0"
	// ticket server
	ticketRepo := common.NewMemoryTicketRepo()
	ticketHandler := ticketimpl.NewTicketService(ticketRepo)
	tLn, err := net.Listen("tcp", anyLoop)
	if err != nil {
		t.Fatalf("ticket listen: %v", err)
	}
	tSrv := ticketservice.NewServer(ticketHandler, server.WithListener(tLn))
	go func() { _ = tSrv.Run() }()
	defer func() { _ = tSrv.Stop(); _ = tLn.Close() }()
	ticketAddr := tLn.Addr().String()
	t.Logf("ticket server addr=%s", ticketAddr)

	// kb server
	kbRepo := kbstore.NewMemoryRepo()
	kbHandler := kbimpl.NewKBService(kbRepo)
	kLn, err := net.Listen("tcp", anyLoop)
	if err != nil {
		t.Fatalf("kb listen: %v", err)
	}
	kbSrv := kbservice.NewServer(kbHandler, server.WithListener(kLn))
	go func() { _ = kbSrv.Run() }()
	defer func() { _ = kbSrv.Stop(); _ = kLn.Close() }()
	kbAddr := kLn.Addr().String()
	t.Logf("kb server addr=%s", kbAddr)

	// ai server
	aiHandler := aiimpl.NewAIService()
	aLn, err := net.Listen("tcp", anyLoop)
	if err != nil {
		t.Fatalf("ai listen: %v", err)
	}
	aiSrv := aiservice.NewServer(aiHandler, server.WithListener(aLn))
	go func() { _ = aiSrv.Run() }()
	defer func() { _ = aiSrv.Stop(); _ = aLn.Close() }()
	aiAddr := aLn.Addr().String()
	t.Logf("ai server addr=%s", aiAddr)

	// rudimentary readiness wait (retry loop instead of fixed sleep)
	deadline := time.Now().Add(2 * time.Second)
	for {
		if time.Now().After(deadline) {
			break
		}
		conn, err := net.DialTimeout("tcp", ticketAddr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(30 * time.Millisecond)
	}

	tCli, err := ticketservice.NewClient("ticket", client.WithHostPorts(ticketAddr))
	if err != nil {
		t.Fatalf("ticket client: %v", err)
	}
	kbCli, err := kbservice.NewClient("kb", client.WithHostPorts(kbAddr))
	if err != nil {
		t.Fatalf("kb client: %v", err)
	}
	aiCli, err := aiservice.NewClient("ai", client.WithHostPorts(aiAddr))
	if err != nil {
		t.Fatalf("ai client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tr, err := tCli.CreateTicket(ctx, &ticket.CreateTicketRequest{Title: "t", Desc: "d"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if tr.GetTicket().GetId() == "" {
		t.Fatalf("empty ticket id")
	}

	doc, err := kbCli.AddDoc(ctx, &kbidl.AddDocRequest{Title: "doc", Content: "abc"})
	if err != nil {
		t.Fatalf("AddDoc: %v", err)
	}
	if doc.GetId() == "" {
		t.Fatalf("empty doc id")
	}
	if _, err = kbCli.Search(ctx, &kbidl.SearchRequest{Query: "doc"}); err != nil {
		t.Fatalf("Search: %v", err)
	}

	eresp, err := aiCli.Embeddings(ctx, &kcommon.EmbeddingRequest{Texts: []string{"x"}, Dim: 4})
	if err != nil {
		t.Fatalf("Embeddings: %v", err)
	}
	if eresp.GetDim() != 4 {
		t.Fatalf("expected dim=4 got %d", eresp.GetDim())
	}
}
