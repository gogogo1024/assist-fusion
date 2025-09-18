//go:build ignore

// Stub for kb RPC service; real Kitex server code will be generated and wired later.
package main

import (
	"log"
	"net"

	"github.com/cloudwego/kitex/server"

	"github.com/gogogo1024/assist-fusion/internal/kb"
	kbservice "github.com/gogogo1024/assist-fusion/kitex_gen/kb/kbservice"
	"github.com/gogogo1024/assist-fusion/services/kb-rpc/handler"
)

func main() {
	repo := kb.NewMemoryRepo() // later: ES backed implementation
	h := handler.NewKBService(repo)
	ln, err := net.Listen("tcp", ":8202")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	svr := kbservice.NewServer(h, server.WithServiceAddr(ln.Addr()))
	log.Println("kb-rpc service listening on :8202")
	if err := svr.Run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
