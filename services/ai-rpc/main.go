//go:build ignore

// Stub for ai RPC service; kitex generated server will replace this.
package main

import (
	"log"
	"net"

	"github.com/cloudwego/kitex/server"
	aiservice "github.com/gogogo1024/assist-fusion/kitex_gen/ai/aiservice"
	"github.com/gogogo1024/assist-fusion/services/ai-rpc/handler"
)

func main() {
	h := handler.NewAIService()
	ln, err := net.Listen("tcp", ":8203")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	svr := aiservice.NewServer(h, server.WithServiceAddr(ln.Addr()))
	log.Println("ai-rpc service listening on :8203")
	if err := svr.Run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
