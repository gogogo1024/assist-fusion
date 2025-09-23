package chain

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	openaiembed "github.com/cloudwego/eino-ext/components/embedding/openai"
	einoaclopenai "github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	baseai "github.com/gogogo1024/assist-fusion/internal/ai"
)

// EmbeddingChain defines minimal embedding ability.
type EmbeddingChain interface {
	Embed(ctx context.Context, texts []string, dim int) ([][]float64, error)
	Provider() string
}

// ChatChain defines minimal chat generation ability.
type ChatChain interface {
	Chat(ctx context.Context, messages []ChatMessage, maxTokens int) (ChatMessage, error)
	Provider() string
	ChatStream(ctx context.Context, messages []ChatMessage, maxTokens int, onDelta func(delta string) bool) (ChatMessage, error)
}

// ChatMessage mirrors ai.ChatMessage for decoupling from IDL package here.
type ChatMessage struct {
	Role    string
	Content string
}

// Provider names
const (
	ProviderMock        = "mock"
	ProviderOpenAI      = "openai"
	emptyContent        = "(empty)"
	errMissingOpenAIKey = "missing OPENAI_API_KEY"
)

// NewEmbeddingChain builds an EmbeddingChain using provider selection.
func NewEmbeddingChain(provider string) EmbeddingChain {
	// Backward compatible wrapper: load env -> override provider arg -> delegate.
	cfg := LoadAIConfigFromEnv()
	cfg.Provider = provider
	return NewEmbeddingChainFromConfig(cfg)
}

// NewChatChain builds a ChatChain using provider selection.
func NewChatChain(provider string) ChatChain {
	cfg := LoadAIConfigFromEnv()
	cfg.Provider = provider
	return NewChatChainFromConfig(cfg)
}

// NewEmbeddingChainFromConfig creates an EmbeddingChain using an explicit AIConfig (env-free for tests).
func NewEmbeddingChainFromConfig(cfg AIConfig) EmbeddingChain {
	if strings.ToLower(cfg.Provider) == ProviderOpenAI {
		if ec, err := newEinoEmbeddingFromConfig(cfg.OpenAIKey, cfg.OpenAIEmbedModel, cfg.OpenAIBaseURL); err == nil {
			return ec
		}
		return newMockEmbedding()
	}
	return newMockEmbedding()
}

// NewChatChainFromConfig creates a ChatChain using an explicit AIConfig (env-free for tests).
func NewChatChainFromConfig(cfg AIConfig) ChatChain {
	if strings.ToLower(cfg.Provider) == ProviderOpenAI {
		if cc, err := newEinoChatFromConfig(cfg.OpenAIKey, cfg.OpenAIChatModel, cfg.OpenAIBaseURL); err == nil {
			return cc
		}
		return newMockChat()
	}
	return newMockChat()
}

// DetectProvider decides provider from env.
func DetectProvider() string {
	p := strings.ToLower(strings.TrimSpace(os.Getenv("AI_PROVIDER")))
	switch p {
	case ProviderOpenAI:
		return ProviderOpenAI
	default:
		return ProviderMock
	}
}

// NOTE: Previously had AI_USE_EINO flag; now eino is mandatory first choice for openai.

// --- Mock implementations (deterministic, lightweight) ---

type mockEmbedding struct{}

func newMockEmbedding() *mockEmbedding { return &mockEmbedding{} }

func (m *mockEmbedding) Embed(ctx context.Context, texts []string, dim int) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, errors.New("no texts")
	}
	return baseai.MockEmbeddings(texts, dim), nil
}

func (m *mockEmbedding) Provider() string { return ProviderMock }

type mockChat struct{}

func newMockChat() *mockChat { return &mockChat{} }

func (m *mockChat) Chat(ctx context.Context, messages []ChatMessage, maxTokens int) (ChatMessage, error) {
	if len(messages) == 0 {
		return ChatMessage{Role: "assistant", Content: emptyContent}, nil
	}
	last := messages[len(messages)-1]
	// simple echo summary
	resp := ChatMessage{Role: "assistant", Content: "echo:" + truncate(last.Content, 200)}
	return resp, nil
}

func (m *mockChat) Provider() string { return ProviderMock }

func (m *mockChat) ChatStream(ctx context.Context, messages []ChatMessage, maxTokens int, onDelta func(string) bool) (ChatMessage, error) {
	if len(messages) == 0 {
		if onDelta != nil {
			onDelta(emptyContent)
		}
		return ChatMessage{Role: "assistant", Content: emptyContent}, nil
	}
	last := messages[len(messages)-1].Content
	out := "echo:" + truncate(last, 200)
	if onDelta != nil {
		seg := len(out) / 3
		if seg == 0 {
			seg = len(out)
		}
		for i := 0; i < len(out); i += seg {
			if !onDelta(out[i:min(i+seg, len(out))]) {
				break
			}
		}
	}
	return ChatMessage{Role: "assistant", Content: out}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// (Legacy OpenAI HTTP implementation removed; eino-ext is now mandatory first & only real path.)

// --- Eino real embedding adapter ---
// Uses eino-ext OpenAI embedder. We keep a lightweight wrapper implementing our EmbeddingChain.

type einoEmbedding struct {
	apiKey   string
	model    string
	baseURL  string
	embedder *openaiembed.Embedder
	curDim   int // dimension configured in current embedder (0 means provider default)
}

// (Deprecated) old env-based constructor removed; use newEinoEmbeddingFromConfig via NewEmbeddingChainFromConfig.

// newEinoEmbeddingFromConfig builds eino embedding using explicit params (test-friendly).
func newEinoEmbeddingFromConfig(key, model, base string) (EmbeddingChain, error) {
	if key == "" {
		return nil, errors.New(errMissingOpenAIKey)
	}
	if model == "" {
		model = "text-embedding-3-small"
	}
	// base may be empty => use provider default
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	emb, err := openaiembed.NewEmbedder(ctx, &openaiembed.EmbeddingConfig{
		APIKey:  key,
		Model:   model,
		BaseURL: base,
		Timeout: 15 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &einoEmbedding{apiKey: key, model: model, baseURL: base, embedder: emb}, nil
}

func (e *einoEmbedding) ensureEmbedderWithDim(ctx context.Context, dim int) error {
	if dim <= 0 || e.curDim == dim {
		return nil
	}
	// recreate embedder with new dimension
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	emb, err := openaiembed.NewEmbedder(ctx, &openaiembed.EmbeddingConfig{
		APIKey:     e.apiKey,
		Model:      e.model,
		BaseURL:    e.baseURL,
		Timeout:    15 * time.Second,
		Dimensions: &dim,
	})
	if err != nil {
		return err
	}
	e.embedder = emb
	e.curDim = dim
	return nil
}

func (e *einoEmbedding) Embed(ctx context.Context, texts []string, dim int) ([][]float64, error) {
	if err := e.ensureEmbedderWithDim(ctx, dim); err != nil {
		return nil, err
	}
	return e.embedder.EmbedStrings(ctx, texts)
}
func (e *einoEmbedding) Provider() string { return ProviderOpenAI }

// --- Eino chat adapter (still legacy delegate; to be replaced with real eino-ext chat component) ---
// --- Eino chat adapter using eino-ext OpenAI client ---
type einoChat struct {
	client *einoaclopenai.Client
	model  string
}

// (Deprecated) old env-based chat constructor removed; use newEinoChatFromConfig via NewChatChainFromConfig.

// newEinoChatFromConfig builds eino chat using explicit params (test-friendly, no env reads).
func newEinoChatFromConfig(key, modelName, base string) (ChatChain, error) {
	if key == "" {
		return nil, errors.New(errMissingOpenAIKey)
	}
	if modelName == "" {
		modelName = "gpt-4o-mini"
	}
	base = strings.TrimRight(base, "/")
	if base == "" { // ensure a sane default; some upstream client paths may not set http.Client if base empty
		base = "https://api.openai.com/v1"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cli, err := einoaclopenai.NewClient(ctx, &einoaclopenai.Config{APIKey: key, Model: modelName, BaseURL: base})
	if err != nil {
		return nil, err
	}
	return &einoChat{client: cli, model: modelName}, nil
}

func (e *einoChat) Provider() string { return ProviderOpenAI }

func (e *einoChat) Chat(ctx context.Context, messages []ChatMessage, maxTokens int) (ret ChatMessage, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in openai chat: %v", r)
		}
	}()
	if len(messages) == 0 {
		return ChatMessage{Role: "assistant", Content: emptyContent}, nil
	}
	mm := toSchemaMessages(messages)
	opts := []model.Option{}
	if maxTokens > 0 {
		opts = append(opts, model.WithMaxTokens(maxTokens))
	}
	resp, genErr := e.client.Generate(ctx, mm, opts...)
	if genErr != nil {
		return ChatMessage{}, genErr
	}
	if resp == nil || resp.Content == "" {
		return ChatMessage{}, errors.New("empty chat content")
	}
	return ChatMessage{Role: string(resp.Role), Content: resp.Content}, nil
}

func (e *einoChat) ChatStream(ctx context.Context, messages []ChatMessage, maxTokens int, onDelta func(string) bool) (ret ChatMessage, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in openai chat stream: %v", r)
		}
	}()
	if len(messages) == 0 {
		return ChatMessage{Role: "assistant", Content: emptyContent}, nil
	}
	mm := toSchemaMessages(messages)
	opts := buildModelOptions(maxTokens)
	stream, sErr := e.client.Stream(ctx, mm, opts...)
	if sErr != nil {
		return ChatMessage{}, sErr
	}
	var full strings.Builder
	consumeEinoStream(stream, &full, onDelta)
	return ChatMessage{Role: "assistant", Content: full.String()}, nil
}

func buildModelOptions(maxTokens int) []model.Option {
	if maxTokens > 0 {
		return []model.Option{model.WithMaxTokens(maxTokens)}
	}
	return nil
}

func consumeEinoStream(stream *schema.StreamReader[*schema.Message], full *strings.Builder, onDelta func(string) bool) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) { // normal end
				return
			}
			return
		}
		if !appendEinoMessage(msg, full, onDelta) {
			return
		}
	}
}

func appendEinoMessage(msg *schema.Message, full *strings.Builder, onDelta func(string) bool) bool {
	if msg == nil || (msg.Content == "" && len(msg.ToolCalls) == 0) {
		return true // nothing to do, keep streaming
	}
	piece := msg.Content
	full.WriteString(piece)
	if onDelta != nil && !onDelta(piece) {
		return false
	}
	return true
}

func toSchemaMessages(in []ChatMessage) []*schema.Message {
	out := make([]*schema.Message, 0, len(in))
	for _, m := range in {
		role := schema.RoleType(m.Role)
		out = append(out, &schema.Message{Role: role, Content: m.Content})
	}
	return out
}

// (Legacy streaming SSE parsing removed.)
