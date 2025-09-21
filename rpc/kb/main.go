package main

import (
	"context"
	"log"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/gogogo1024/assist-fusion/internal/kb"
	"github.com/gogogo1024/assist-fusion/internal/kitexconf"
	kbservice "github.com/gogogo1024/assist-fusion/kitex_gen/kb/kbservice"
	kbimpl "github.com/gogogo1024/assist-fusion/rpc/kb/impl"
)

func main() {
	cfg, err := kitexconf.Load("kb")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := kitexconf.InitLogger(cfg); err != nil {
		log.Printf("init logger failed (fallback std log only): %v", err)
	}
	repo := kb.NewMemoryRepo()
	h := kbimpl.NewKBService(repo)
	opts, err := kitexconf.BuildServerOptions(cfg)
	if err != nil {
		klog.Fatalf("build opts: %v", err)
	}
	hooks := kitexconf.InitRuntime(context.Background(), cfg)
	defer hooks.Shutdown(context.Background())
	svr := kbservice.NewServer(h, opts...)
	klog.Infof("kb service starting env=%s addr=%s config=%s", cfg.Env, cfg.Kitex.Address, cfg.RawPath)
	if err := svr.Run(); err != nil {
		klog.Errorf("server stopped: %v", err)
	}
}
