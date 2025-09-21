package rpc

import (
	"sync"

	"github.com/cloudwego/kitex/client"
	"github.com/gogogo1024/assist-fusion/common/clientsuite"
	"github.com/gogogo1024/assist-fusion/internal/common"
	"github.com/gogogo1024/assist-fusion/kitex_gen/ai/aiservice"
	"github.com/gogogo1024/assist-fusion/kitex_gen/kb/kbservice"
	"github.com/gogogo1024/assist-fusion/kitex_gen/ticket/ticketservice"
)

var (
	TicketClient ticketservice.Client
	KBClient     kbservice.Client
	AIClient     aiservice.Client

	once    sync.Once
	initErr error
)

// Init initializes kitex clients with consul resolver + tracing etc via clientsuite.
func Init(cfg *common.Config) error {
	once.Do(func() {
		suite := clientsuite.CommonGrpcClientSuite{
			RegistryAddr:       cfg.RegistryAddr,
			CurrentServiceName: "gateway", // service name for tracing peer info
		}
		opts := []client.Option{client.WithSuite(suite)}

		TicketClient, initErr = ticketservice.NewClient("ticket-rpc", opts...)
		if initErr != nil {
			return
		}
		KBClient, initErr = kbservice.NewClient("kb-rpc", opts...)
		if initErr != nil {
			return
		}
		AIClient, initErr = aiservice.NewClient("ai-rpc", opts...)
	})
	return initErr
}
