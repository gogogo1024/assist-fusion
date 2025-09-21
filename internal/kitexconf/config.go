package kitexconf

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config represents the aggregated runtime configuration for a kitex service.
// It mirrors (and simplifies) the structure observed in the external cart service
// you provided: conf/<env>/conf.yaml with sections: kitex, mysql, redis, registry.
type Config struct {
	Env      string         `yaml:"-"`
	Kitex    KitexConfig    `yaml:"kitex"`
	MySQL    MySQLConfig    `yaml:"mysql"`
	Redis    RedisConfig    `yaml:"redis"`
	Registry RegistryConfig `yaml:"registry"`
	// RawPath records the loaded file path for diagnostics.
	RawPath string `yaml:"-"`
}

type KitexConfig struct {
	Service       string `yaml:"service"`
	Address       string `yaml:"address"`
	LogLevel      string `yaml:"log_level"`
	LogFileName   string `yaml:"log_file_name"`
	LogMaxSize    int    `yaml:"log_max_size"` // MB
	LogMaxBackups int    `yaml:"log_max_backups"`
	LogMaxAge     int    `yaml:"log_max_age"` // days
	MetricsPort   string `yaml:"metrics_port"`
}

type MySQLConfig struct {
	DSN string `yaml:"dsn"`
}

type RedisConfig struct {
	Address  string `yaml:"address"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type RegistryConfig struct {
	RegistryAddress []string `yaml:"registry_address"`
	Username        string   `yaml:"username"`
	Password        string   `yaml:"password"`
}

var (
	loaded    *Config
	loadOnce  sync.Once
	loadError error
)

// Load loads the config for a given service name (e.g. "ticket", "kb", "ai").
// It follows the directory convention: rpc/<service>/conf/<env>/conf.yaml
// Environment precedence:
//  1. Explicit CONF_FILE (absolute or relative path)
//  2. GO_ENV (values like dev/test/online), default = test
//  3. Fallback to test if the requested env file does not exist
//
// A singleton is cached; subsequent calls return the same pointer.
func Load(service string) (*Config, error) {
	loadOnce.Do(func() {
		loaded, loadError = loadInternal(service)
	})
	return loaded, loadError
}

func loadInternal(service string) (*Config, error) {
	if service == "" {
		return nil, errors.New("service name required")
	}
	// 1. explicit file
	if explicit := os.Getenv("CONF_FILE"); explicit != "" {
		if cfg, err := parseFile(explicit); err == nil {
			cfg.Env = envValue()
			cfg.RawPath = explicit
			postProcess(cfg, service)
			return cfg, nil
		} else {
			return nil, fmt.Errorf("load explicit CONF_FILE failed: %w", err)
		}
	}
	env := envValue()
	// try desired env
	candidate := filepath.Join("rpc", service, "conf", env, "conf.yaml")
	if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
		// fallback to test
		fallback := filepath.Join("rpc", service, "conf", "test", "conf.yaml")
		if _, ferr := os.Stat(fallback); ferr == nil {
			candidate = fallback
			env = "test"
		} else {
			return nil, fmt.Errorf("config file not found: %s (no fallback)", candidate)
		}
	}
	cfg, err := parseFile(candidate)
	if err != nil {
		return nil, err
	}
	cfg.Env = env
	cfg.RawPath = candidate
	postProcess(cfg, service)
	return cfg, nil
}

func envValue() string {
	if v := os.Getenv("GO_ENV"); v != "" {
		return v
	}
	return "test"
}

func parseFile(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := new(Config)
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// postProcess applies sane defaults and environment overrides.
func postProcess(c *Config, service string) {
	// default service name if empty
	if c.Kitex.Service == "" {
		c.Kitex.Service = service
	}
	// default address
	if c.Kitex.Address == "" {
		c.Kitex.Address = defaultAddressFor(service)
	}
	// env override for log level
	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		c.Kitex.LogLevel = lvl
	}
	// service specific address override e.g. TICKET_ADDR
	upper := strings.ToUpper(service) + "_ADDR"
	if addr := os.Getenv(upper); addr != "" {
		c.Kitex.Address = addr
	}
	// defaults for log rotation if zero
	if c.Kitex.LogMaxSize == 0 {
		c.Kitex.LogMaxSize = 50
	}
	if c.Kitex.LogMaxBackups == 0 {
		c.Kitex.LogMaxBackups = 5
	}
	if c.Kitex.LogMaxAge == 0 {
		c.Kitex.LogMaxAge = 14
	}
	if c.Kitex.LogFileName == "" {
		// put log under service specific dir
		c.Kitex.LogFileName = filepath.Join("rpc", service, "log", "kitex.log")
	}
}

func defaultAddressFor(service string) string {
	// simple mapping (adjust if needed)
	switch service {
	case "ticket":
		return ":8201"
	case "kb":
		return ":8202"
	case "ai":
		return ":8203"
	default:
		return ":0" // let OS choose
	}
}
