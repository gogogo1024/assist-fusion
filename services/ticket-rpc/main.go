// Ticket RPC service entrypoint
package main

import (
	"log"
	"net"

	"github.com/cloudwego/kitex/server"
	"github.com/gogogo1024/assist-fusion/internal/common"
	ticketservice "github.com/gogogo1024/assist-fusion/kitex_gen/ticket/ticketservice"
	"github.com/gogogo1024/assist-fusion/services/ticket-rpc/handler"
)

func main() {
	repo := common.NewMemoryTicketRepo() // Phase A
	h := handler.NewTicketService(repo)
	ln, err := net.Listen("tcp", ":8201")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	svr := ticketservice.NewServer(h, server.WithServiceAddr(ln.Addr()))
	log.Println("ticket-rpc service listening on :8201")
	if err := svr.Run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
