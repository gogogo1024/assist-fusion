package common

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr     string
	DBDSN        string
	VectorDSN    string
	AIProvider   string
	AIAPIKey     string
	OtelEndpoint string
	// Consul registry address (single)
	RegistryAddr string
	// KB backend selection: "memory" (default) or "es"
	KBBackend string
	// Elasticsearch settings for KB when KBBackend=="es"
	ESAddrs    []string
	ESIndex    string
	ESUsername string
	ESPassword string
	// Feature flags
	FeatureRPC bool
	// Downstream RPC addresses (host:port)
	TicketRPCAddr string
	KBRPCAddr     string
	AIRPCAddr     string
}

func LoadConfig() *Config {
	esAddrs := getenv("ES_ADDRS", "")
	var addrs []string
	if esAddrs != "" {
		for _, p := range strings.Split(esAddrs, ",") {
			v := strings.TrimSpace(p)
			if v != "" {
				addrs = append(addrs, v)
			}
		}
	}
	return &Config{
		HTTPAddr:      getenv("HTTP_ADDR", ":8080"),
		DBDSN:         getenv("DB_DSN", ""),
		VectorDSN:     getenv("VECTOR_DSN", ""),
		AIProvider:    getenv("AI_PROVIDER", "mock"),
		AIAPIKey:      getenv("AI_API_KEY", ""),
		OtelEndpoint:  getenv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		RegistryAddr:  getenv("CONSUL_ADDR", "127.0.0.1:8500"),
		KBBackend:     getenv("KB_BACKEND", "memory"),
		ESAddrs:       addrs,
		ESIndex:       getenv("ES_INDEX", "kb_docs"),
		ESUsername:    getenv("ES_USERNAME", ""),
		ESPassword:    getenv("ES_PASSWORD", ""),
		FeatureRPC:    strings.EqualFold(getenv("FEATURE_RPC", "off"), "on"),
		TicketRPCAddr: getenv("TICKET_RPC_ADDR", "127.0.0.1:8201"),
		KBRPCAddr:     getenv("KB_RPC_ADDR", "127.0.0.1:8202"),
		AIRPCAddr:     getenv("AI_RPC_ADDR", "127.0.0.1:8203"),
	}
}

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

// EsAddressesOrDefault returns configured ES addresses or a local default.
func (c *Config) EsAddressesOrDefault() []string {
	if len(c.ESAddrs) > 0 {
		return c.ESAddrs
	}
	// default to local single node
	return []string{"http://localhost:9200"}
}
