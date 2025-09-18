//go:build ignore

package main

import (
	"log"

	ticket "github.com/gogogo1024/assist-fusion/kitex_gen/ticket/ticketservice"
)

func main() {
	svr := ticket.NewServer(new(TicketServiceImpl))

	err := svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
