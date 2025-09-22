package rpc

import (
	"os"
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
		// Test / fallback mode: allow disabling consul resolver entirely and use direct host:ports.
		// Triggered when DISABLE_CONSUL=1 (or empty registry address) so integration tests do not require a registry.
		disableConsul := os.Getenv("DISABLE_CONSUL") == "1" || cfg.RegistryAddr == "" || cfg.RegistryAddr == "0"
		if disableConsul {
			TicketClient, initErr = ticketservice.NewClient("ticket", client.WithHostPorts(cfg.TicketRPCAddr))
			if initErr != nil {
				return
			}
			KBClient, initErr = kbservice.NewClient("kb", client.WithHostPorts(cfg.KBRPCAddr))
			if initErr != nil {
				return
			}
			AIClient, initErr = aiservice.NewClient("ai", client.WithHostPorts(cfg.AIRPCAddr))
			return
		}

		suite := clientsuite.CommonGrpcClientSuite{
			RegistryAddr:       cfg.RegistryAddr,
			CurrentServiceName: "gateway", // service name for tracing peer info
		}
		opts := []client.Option{client.WithSuite(suite)}

		TicketClient, initErr = ticketservice.NewClient("ticket", opts...)
		if initErr != nil {
			return
		}
		KBClient, initErr = kbservice.NewClient("kb", opts...)
		if initErr != nil {
			return
		}
		AIClient, initErr = aiservice.NewClient("ai", opts...)
	})
	return initErr
}
