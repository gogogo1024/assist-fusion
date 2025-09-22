package chain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
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
	ProviderMock      = "mock"
	ProviderOpenAI    = "openai"
	bearerPrefix      = "Bearer "
	headerContentType = "Content-Type"
	mimeJSON          = "application/json"
	emptyContent      = "(empty)"
)

// NewEmbeddingChain builds an EmbeddingChain using provider selection.
func NewEmbeddingChain(provider string) EmbeddingChain {
	// Eino flag only meaningful for openai provider currently.
	if provider == ProviderOpenAI && useEino() {
		if ec, err := newEinoEmbedding(); err == nil {
			return ec
		}
		// fallthrough to legacy
	}
	switch provider {
	case ProviderOpenAI:
		if c, err := newOpenAIEmbedding(); err == nil {
			return c
		}
		// silent fallback to mock (metrics handled at service layer)
		return newMockEmbedding()
	default:
		return newMockEmbedding()
	}
}

// NewChatChain builds a ChatChain using provider selection.
func NewChatChain(provider string) ChatChain {
	if provider == ProviderOpenAI && useEino() {
		if cc, err := newEinoChat(); err == nil {
			return cc
		}
		// fallback to legacy
	}
	switch provider {
	case ProviderOpenAI:
		if c, err := newOpenAIChat(); err == nil {
			return c
		}
		return newMockChat()
	default:
		return newMockChat()
	}
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

// useEino returns true when AI_USE_EINO=1 (string compare) enabling eino adapter path.
func useEino() bool { return strings.TrimSpace(os.Getenv("AI_USE_EINO")) == "1" }

// --- Mock implementations (deterministic, lightweight) ---

type mockEmbedding struct{}

func newMockEmbedding() *mockEmbedding { return &mockEmbedding{} }

func (m *mockEmbedding) Embed(ctx context.Context, texts []string, dim int) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, errors.New("no texts")
	}
	if dim <= 0 {
		dim = 32
	}
	out := make([][]float64, len(texts))
	for i, t := range texts {
		h := sha256.Sum256([]byte(t))
		vec := make([]float64, dim)
		for d := 0; d < dim; d++ {
			b := h[d%len(h)]
			vec[d] = float64(int(b)%200-100) / 100.0
		}
		out[i] = vec
	}
	return out, nil
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

// --- OpenAI adapter (real REST integration minimal) ---

type openAIEmbedding struct {
	apiKey  string
	model   string
	timeout time.Duration
	baseURL string
	httpc   *http.Client
}

func newOpenAIEmbedding() (EmbeddingChain, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, errors.New("missing OPENAI_API_KEY")
	}
	model := os.Getenv("OPENAI_EMBED_MODEL")
	if model == "" {
		model = "text-embedding-3-small"
	}
	base := strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	return &openAIEmbedding{apiKey: key, model: model, timeout: 15 * time.Second, baseURL: base, httpc: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (o *openAIEmbedding) Provider() string { return ProviderOpenAI }

type openAIEmbReq struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions *int     `json:"dimensions,omitempty"`
}
type openAIEmbResp struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (o *openAIEmbedding) Embed(ctx context.Context, texts []string, dim int) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, errors.New("no texts")
	}
	reqBody := openAIEmbReq{Model: o.model, Input: texts}
	if dim > 0 {
		reqBody.Dimensions = &dim
	}
	b, _ := json.Marshal(reqBody)
	url := o.baseURL + "/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", bearerPrefix+o.apiKey)
	httpReq.Header.Set(headerContentType, mimeJSON)
	resp, err := o.httpc.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("openai embeddings http %d: %s", resp.StatusCode, truncate(string(rb), 180))
	}
	var parsed openAIEmbResp
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return nil, fmt.Errorf("decode embeddings: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("openai embeddings error: %s", parsed.Error.Message)
	}
	if len(parsed.Data) == 0 {
		return nil, errors.New("empty embeddings data")
	}
	out := make([][]float64, len(parsed.Data))
	for i, d := range parsed.Data {
		out[i] = d.Embedding
	}
	// If user requested dim and provider returned larger, slice; if smaller, pad zeros.
	if dim > 0 {
		for i := range out {
			if len(out[i]) > dim {
				out[i] = out[i][:dim]
			} else if len(out[i]) < dim {
				padded := make([]float64, dim)
				copy(padded, out[i])
				out[i] = padded
			}
		}
	}
	return out, nil
}

type openAIChat struct {
	apiKey  string
	model   string
	timeout time.Duration
	baseURL string
	httpc   *http.Client
}

func newOpenAIChat() (ChatChain, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, errors.New("missing OPENAI_API_KEY")
	}
	model := os.Getenv("OPENAI_CHAT_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	base := strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	return &openAIChat{apiKey: key, model: model, timeout: 30 * time.Second, baseURL: base, httpc: &http.Client{Timeout: 30 * time.Second}}, nil
}

func (o *openAIChat) Provider() string { return ProviderOpenAI }

type openAIChatReq struct {
	Model     string              `json:"model"`
	Messages  []openAIChatMessage `json:"messages"`
	MaxTokens *int                `json:"max_tokens,omitempty"`
	Stream    bool                `json:"stream,omitempty"`
}

// --- Eino adapter stubs (Phase 1) ---
// For now they simply delegate to existing OpenAI implementations; later can be replaced by real eino components.

type einoEmbedding struct{ inner EmbeddingChain }

func newEinoEmbedding() (EmbeddingChain, error) {
	// Reuse openai embedding path; if fails propagate error so fallback applies.
	c, err := newOpenAIEmbedding()
	if err != nil { return nil, err }
	return &einoEmbedding{inner: c}, nil
}

func (e *einoEmbedding) Embed(ctx context.Context, texts []string, dim int) ([][]float64, error) {
	return e.inner.Embed(ctx, texts, dim)
}
func (e *einoEmbedding) Provider() string { return ProviderOpenAI }

type einoChat struct{ inner ChatChain }

func newEinoChat() (ChatChain, error) {
	c, err := newOpenAIChat()
	if err != nil { return nil, err }
	return &einoChat{inner: c}, nil
}

func (e *einoChat) Chat(ctx context.Context, messages []ChatMessage, maxTokens int) (ChatMessage, error) {
	return e.inner.Chat(ctx, messages, maxTokens)
}
func (e *einoChat) ChatStream(ctx context.Context, messages []ChatMessage, maxTokens int, onDelta func(string) bool) (ChatMessage, error) {
	return e.inner.ChatStream(ctx, messages, maxTokens, onDelta)
}
func (e *einoChat) Provider() string { return ProviderOpenAI }
type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type openAIChatResp struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (o *openAIChat) Chat(ctx context.Context, messages []ChatMessage, maxTokens int) (ChatMessage, error) {
	if len(messages) == 0 {
		return ChatMessage{Role: "assistant", Content: emptyContent}, nil
	}
	ms := make([]openAIChatMessage, 0, len(messages))
	for _, m := range messages {
		ms = append(ms, openAIChatMessage(m))
	}
	reqPayload := openAIChatReq{Model: o.model, Messages: ms}
	if maxTokens > 0 {
		reqPayload.MaxTokens = &maxTokens
	}
	b, _ := json.Marshal(reqPayload)
	url := o.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return ChatMessage{}, err
	}
	httpReq.Header.Set("Authorization", bearerPrefix+o.apiKey)
	httpReq.Header.Set(headerContentType, mimeJSON)
	resp, err := o.httpc.Do(httpReq)
	if err != nil {
		return ChatMessage{}, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return ChatMessage{}, fmt.Errorf("openai chat http %d: %s", resp.StatusCode, truncate(string(rb), 180))
	}
	var parsed openAIChatResp
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return ChatMessage{}, fmt.Errorf("decode chat: %w", err)
	}
	if parsed.Error != nil {
		return ChatMessage{}, fmt.Errorf("openai chat error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return ChatMessage{}, errors.New("no chat choices")
	}
	c := parsed.Choices[0].Message
	if c.Content == "" {
		return ChatMessage{}, errors.New("empty chat content")
	}
	return ChatMessage(c), nil
}

// ChatStream implements streaming chat completions using SSE style data: lines.
func (o *openAIChat) ChatStream(ctx context.Context, messages []ChatMessage, maxTokens int, onDelta func(string) bool) (ChatMessage, error) {
    if len(messages) == 0 { return ChatMessage{Role: "assistant", Content: "(empty)"}, nil }
    httpReq, err := o.buildStreamRequest(ctx, messages, maxTokens)
    if err != nil { return ChatMessage{}, err }
    resp, err := o.httpc.Do(httpReq)
    if err != nil { return ChatMessage{}, err }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        rb, _ := io.ReadAll(resp.Body)
        return ChatMessage{}, fmt.Errorf("openai chat stream http %d: %s", resp.StatusCode, truncate(string(rb), 160))
    }
    return o.consumeStream(resp.Body, onDelta)
}

func (o *openAIChat) buildStreamRequest(ctx context.Context, messages []ChatMessage, maxTokens int) (*http.Request, error) {
    ms := make([]openAIChatMessage, 0, len(messages))
    for _, m := range messages { ms = append(ms, openAIChatMessage(m)) }
    reqPayload := openAIChatReq{Model: o.model, Messages: ms, Stream: true}
    if maxTokens > 0 { reqPayload.MaxTokens = &maxTokens }
    b, _ := json.Marshal(reqPayload)
    url := o.baseURL + "/chat/completions"
    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
    if err != nil { return nil, err }
    httpReq.Header.Set("Authorization", bearerPrefix+o.apiKey)
    httpReq.Header.Set(headerContentType, mimeJSON)
    return httpReq, nil
}

func (o *openAIChat) consumeStream(body io.Reader, onDelta func(string) bool) (ChatMessage, error) {
	var full strings.Builder
	dec := newLineReader(body)
	for {
		line, eof, rerr := dec.ReadLine()
		if rerr != nil { return ChatMessage{}, rerr }
		stop := handleOpenAILine(line, eof, &full, onDelta)
		if stop { break }
	}
	return ChatMessage{Role: "assistant", Content: full.String()}, nil
}

// handleOpenAILine processes a single SSE line, returns true if streaming should stop.
func handleOpenAILine(line string, eof bool, full *strings.Builder, onDelta func(string) bool) bool {
	if len(line) == 0 && eof { return true }
	if !strings.HasPrefix(line, "data:") { return eof }
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if payload == "[DONE]" { return true }
	piece := parseOpenAIStreamChunk(payload)
	if piece == "" { return eof }
	full.WriteString(piece)
	if onDelta != nil && !onDelta(piece) { return true }
	return eof
}

func parseOpenAIStreamChunk(payload string) string {
    var chunk struct { Choices []struct { Delta struct { Content string `json:"content"` } `json:"delta"` } `json:"choices"` }
    if err := json.Unmarshal([]byte(payload), &chunk); err != nil { return "" }
    if len(chunk.Choices) == 0 { return "" }
    return chunk.Choices[0].Delta.Content
}

// lineReader reads lines from io.Reader (simple, not optimized for huge streams).
type lineReader struct {
	src io.Reader
	buf []byte
}

func newLineReader(r io.Reader) *lineReader { return &lineReader{src: r, buf: make([]byte, 0, 4096)} }

func (lr *lineReader) ReadLine() (string, bool, error) {
	tmp := make([]byte, 512)
	for {
		if i := bytes.IndexByte(lr.buf, '\n'); i >= 0 {
			line := string(lr.buf[:i])
			lr.buf = lr.buf[i+1:]
			return line, false, nil
		}
		n, err := lr.src.Read(tmp)
		if n > 0 {
			lr.buf = append(lr.buf, tmp[:n]...)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(lr.buf) == 0 {
					return "", true, nil
				}
				line := string(lr.buf)
				lr.buf = lr.buf[:0]
				return line, true, nil
			}
			return "", false, err
		}
	}
}
