package kitexconf

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/kitex/pkg/klog"
)

// InitLogger sets basic klog level and optionally redirects output to a file.
// (Simplified version – can be enhanced later with zap if needed.)
func InitLogger(cfg *Config) error {
	if cfg == nil {
		return nil
	}
	lvl := parseLevel(cfg.Kitex.LogLevel)
	klog.SetLevel(lvl)
	if fn := strings.TrimSpace(cfg.Kitex.LogFileName); fn != "" {
		if dir := filepath.Dir(fn); dir != "." {
			_ = os.MkdirAll(dir, 0o755)
		}
		f, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return err
		}
		klog.SetOutput(f)
	}
	return nil
}

func parseLevel(l string) klog.Level {
	switch strings.ToLower(l) {
	case "debug":
		return klog.LevelDebug
	case "warn":
		return klog.LevelWarn
	case "error":
		return klog.LevelError
	case "fatal":
		return klog.LevelFatal
	default:
		return klog.LevelInfo
	}
}
