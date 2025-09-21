package main

import (
	"context"
	"log"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/gogogo1024/assist-fusion/internal/common"
	"github.com/gogogo1024/assist-fusion/internal/kitexconf"
	"github.com/gogogo1024/assist-fusion/kitex_gen/ticket/ticketservice"
	ticketimpl "github.com/gogogo1024/assist-fusion/rpc/ticket/impl"
)

func main() {
	cfg, err := kitexconf.Load("ticket")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := kitexconf.InitLogger(cfg); err != nil {
		log.Printf("init logger failed (fallback std log only): %v", err)
	}
	repo := common.NewMemoryTicketRepo()
	h := ticketimpl.NewTicketService(repo)
	opts, err := kitexconf.BuildServerOptions(cfg)
	if err != nil {
		klog.Fatalf("build opts: %v", err)
	}
	hooks := kitexconf.InitRuntime(context.Background(), cfg)
	defer hooks.Shutdown(context.Background())
	svr := ticketservice.NewServer(h, opts...)
	klog.Infof("ticket service starting env=%s addr=%s config=%s", cfg.Env, cfg.Kitex.Address, cfg.RawPath)
	if err := svr.Run(); err != nil {
		klog.Errorf("server stopped: %v", err)
	}
}
