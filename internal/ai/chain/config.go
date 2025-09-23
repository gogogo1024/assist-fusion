package chain

import (
	"os"
	"strings"
)

// AIConfig centralizes all AI related runtime configuration.
type AIConfig struct {
	Provider         string
	OpenAIKey        string
	OpenAIEmbedModel string
	OpenAIChatModel  string
	OpenAIBaseURL    string
}

// LoadAIConfigFromEnv loads configuration from environment variables.
// This is the only place we touch os.Getenv so that business logic is decoupled from env lookups.
func LoadAIConfigFromEnv() AIConfig {
	cfg := AIConfig{
		Provider:         getEnvDefault("AI_PROVIDER", "mock"),
		OpenAIKey:        os.Getenv("OPENAI_API_KEY"),
		OpenAIEmbedModel: getEnvDefault("OPENAI_EMBED_MODEL", "text-embedding-3-small"),
		OpenAIChatModel:  getEnvDefault("OPENAI_CHAT_MODEL", "gpt-4o-mini"),
		OpenAIBaseURL:    strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/"),
	}
	if cfg.OpenAIBaseURL == "" { // unified default so both embedding & chat have consistent base
		cfg.OpenAIBaseURL = "https://api.openai.com/v1"
	}
	return cfg
}

func getEnvDefault(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}
