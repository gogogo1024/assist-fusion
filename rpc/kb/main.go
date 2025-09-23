package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/gogogo1024/assist-fusion/internal/kb"
	"github.com/gogogo1024/assist-fusion/internal/kb/esrepo"
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
	backend := strings.ToLower(os.Getenv("KB_BACKEND"))
	var repo kb.Repo
	var backendLabel string
	var analyzerMode string
	if backend == "es" {
		// Build ES repo
		addrsEnv := os.Getenv("ES_ADDRS")
		var addrs []string
		if addrsEnv != "" {
			for _, p := range strings.Split(addrsEnv, ",") {
				v := strings.TrimSpace(p)
				if v != "" {
					addrs = append(addrs, v)
				}
			}
		}
		index := os.Getenv("ES_INDEX")
		r, err := esrepo.New(esrepo.Config{Addresses: addrs, Index: index, Username: os.Getenv("ES_USERNAME"), Password: os.Getenv("ES_PASSWORD")})
		if err != nil {
			klog.Fatalf("init es repo failed: %v", err)
		}
		// probe analyzer mode lazily; ignore errors (fall back unknown)
		if info, ierr := r.Info(context.Background()); ierr == nil {
			if m, ok := info["mode"].(string); ok {
				analyzerMode = m
			}
		}
		repo = r
		backendLabel = "es"
	} else {
		repo = kb.NewMemoryRepo()
		backendLabel = "memory"
	}
	h := kbimpl.NewKBService(repo, kbimpl.WithBackend(backendLabel), kbimpl.WithAnalyzerMode(analyzerMode))
	opts, err := kitexconf.BuildServerOptions(cfg)
	if err != nil {
		klog.Fatalf("build opts: %v", err)
	}
	hooks := kitexconf.InitRuntime(context.Background(), cfg)
	defer hooks.Shutdown(context.Background())
	svr := kbservice.NewServer(h, opts...)
	klog.Infof("kb service starting env=%s addr=%s config=%s backend=%s analyzer=%s", cfg.Env, cfg.Kitex.Address, cfg.RawPath, backendLabel, analyzerMode)
	if err := svr.Run(); err != nil {
		klog.Errorf("server stopped: %v", err)
	}
}
