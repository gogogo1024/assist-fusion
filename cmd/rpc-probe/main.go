package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"syscall"
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

func mustTicketClient(addr string) ticketservice.Client {
	c, err := ticketservice.NewClient("ticket-rpc", client.WithHostPorts(addr))
	if err != nil {
		log.Fatalf("create ticket client %s: %v", addr, err)
	}
	return c
}
func mustKBClient(addr string) kbservice.Client {
	c, err := kbservice.NewClient("kb-rpc", client.WithHostPorts(addr))
	if err != nil {
		log.Fatalf("create kb client %s: %v", addr, err)
	}
	return c
}
func mustAIClient(addr string) aiservice.Client {
	c, err := aiservice.NewClient("ai-rpc", client.WithHostPorts(addr))
	if err != nil {
		log.Fatalf("create ai client %s: %v", addr, err)
	}
	return c
}

func envOr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func main() {
	var (
		timeout    = flag.Duration("timeout", 10*time.Second, "overall timeout")
		waitReady  = flag.Duration("wait-start", 4*time.Second, "max wait for inline servers ready")
		withInline = flag.Bool("inline", false, "start servers inline instead of using existing ones (overrides START_INLINE env)")
	)
	flag.Parse()
	startInline := *withInline || os.Getenv("START_INLINE") == "1"

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	ticketAddr := envOr("TICKET_RPC_ADDR", ":8201")
	kbAddr := envOr("KB_RPC_ADDR", ":8202")
	aiAddr := envOr("AI_RPC_ADDR", ":8203")

	if startInline {
		ticketAddr = startTicketServer()
		kbAddr = startKBServer()
		aiAddr = startAIServer()
		// readiness loop: attempt tcp connect until success or deadline
		addrs := []string{ticketAddr, kbAddr, aiAddr}
		deadline := time.Now().Add(*waitReady)
		for _, a := range addrs {
			if err := waitPortReady(a, deadline); err != nil {
				log.Fatalf("server %s not ready: %v", a, err)
			}
		}
	}

	log.Printf("Probe using ticket=%s kb=%s ai=%s inline=%v timeout=%s", ticketAddr, kbAddr, aiAddr, startInline, timeout.String())

	tCli := mustTicketClient(ticketAddr)
	kCli := mustKBClient(kbAddr)
	aCli := mustAIClient(aiAddr)

	// 1. Ticket Create
	tResp, err := tCli.CreateTicket(ctx, &ticket.CreateTicketRequest{Title: "probe ticket", Desc: "desc"})
	if err != nil {
		log.Fatalf("CreateTicket: %v", err)
	}
	fmt.Println("Ticket.Create =>", tResp.GetTicket().GetId(), tResp.GetTicket().GetStatus())

	// 2. Ticket List
	listResp, err := tCli.ListTickets(ctx, &ticket.ListTicketsRequest{})
	if err != nil {
		log.Fatalf("ListTickets: %v", err)
	}
	fmt.Println("Ticket.List count=", len(listResp.GetTickets()))

	// 3. KB AddDoc
	doc, err := kCli.AddDoc(ctx, &kbidl.AddDocRequest{Title: "probe doc", Content: "hello world"})
	if err != nil {
		log.Fatalf("KB AddDoc: %v", err)
	}
	fmt.Println("KB.AddDoc =>", doc.GetId())

	// 4. KB Search
	sResp, err := kCli.Search(ctx, &kbidl.SearchRequest{Query: "probe"})
	if err != nil {
		log.Fatalf("KB Search: %v", err)
	}
	fmt.Println("KB.Search returned=", sResp.GetReturned())

	// 5. AI Embeddings
	eResp, err := aCli.Embeddings(ctx, &kcommon.EmbeddingRequest{Texts: []string{"hello"}, Dim: 8})
	if err != nil {
		log.Fatalf("Embeddings: %v", err)
	}
	fmt.Println("AI.Embeddings vectors=", len(eResp.GetVectors()), "dim=", eResp.GetDim())

	// 6. AI Chat (expect error)
	if _, err = aCli.Chat(ctx, nil); err != nil {
		fmt.Println("AI.Chat expected error:", err)
	} else {
		fmt.Println("AI.Chat unexpectedly succeeded")
	}
}

// --- inline server helpers ---
// find a free TCP port by binding :0 then closing
func freePort() string {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("grab free port: %v", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	p := addr.Port
	_ = ln.Close()
	return fmt.Sprintf(":%d", p)
}
func startTicketServer() string {
	repo := common.NewMemoryTicketRepo()
	h := ticketimpl.NewTicketService(repo)
	addr := freePort()
	svr := ticketservice.NewServer(h, server.WithServiceAddr(&net.TCPAddr{Port: mustPort(addr)}))
	go func() { _ = svr.Run() }()
	return normalizeLocal(addr)
}
func startKBServer() string {
	repo := kbstore.NewMemoryRepo()
	h := kbimpl.NewKBService(repo)
	addr := freePort()
	svr := kbservice.NewServer(h, server.WithServiceAddr(&net.TCPAddr{Port: mustPort(addr)}))
	go func() { _ = svr.Run() }()
	return normalizeLocal(addr)
}
func startAIServer() string {
	h := aiimpl.NewAIService()
	addr := freePort()
	svr := aiservice.NewServer(h, server.WithServiceAddr(&net.TCPAddr{Port: mustPort(addr)}))
	go func() { _ = svr.Run() }()
	return normalizeLocal(addr)
}

func mustPort(addr string) int {
	// addr like :12345
	var p int
	if _, err := fmt.Sscanf(addr, ":%d", &p); err != nil || p <= 0 {
		log.Fatalf("parse port from %s: %v", addr, err)
	}
	return p
}
func normalizeLocal(addr string) string { return "127.0.0.1" + addr }

// waitPortReady tries to connect until success or deadline.
func waitPortReady(addr string, deadline time.Time) error {
	// Normalize addr (Kitex may output 127.0.0.1:X or [::]:X) we just try as-is.
	for {
		if time.Now().After(deadline) {
			return errors.New("timeout waiting for " + addr)
		}
		// quick parse just to ensure format
		if _, err := netip.ParseAddrPort(addr); err != nil {
			// if parse fails, prepend 127.0.0.1 if only :port pattern
			if addr[0] == ':' {
				addr = "127.0.0.1" + addr
			}
		}
		d := net.Dialer{Timeout: 120 * time.Millisecond}
		c, err := d.Dial("tcp", addr)
		if err == nil {
			_ = c.Close()
			return nil
		}
		// ignore temporary
		if ne, ok := err.(interface{ Temporary() bool }); ok && ne.Temporary() {
			time.Sleep(60 * time.Millisecond)
			continue
		}
		// backoff small
		time.Sleep(80 * time.Millisecond)
	}
}

// graceful handling of SIGINT so inline servers exit cleanly
func init() {
	go func() {
		sigCh := make(chan os.Signal, 2)
		// use direct syscall values to avoid extra import of signal.Notify for brevity
		// (we still import syscall). If needed can expand.
		// This is a minimal placeholder—Kitex servers exit on process signal anyway.
		_ = sigCh
		_ = syscall.Getpid()
	}()
}
