package observability

import (
	"fmt"
	"sync/atomic"
)

var (
	TicketCreated    atomic.Int64
	TicketAssigned   atomic.Int64
	TicketEscalated  atomic.Int64
	TicketResolved   atomic.Int64
	TicketReopened   atomic.Int64
	KBDocCreated     atomic.Int64
	KBDocUpdated     atomic.Int64
	KBDocDeleted     atomic.Int64
	KBSearchRequests atomic.Int64
	KBSearchHits     atomic.Int64
	AIEmbeddingCalls atomic.Int64

	// AI provider granular counters
	AIEmbeddingSuccessMock  atomic.Int64
	AIEmbeddingFallbackMock atomic.Int64
	AIEmbeddingError        atomic.Int64
	AIChatSuccessMock       atomic.Int64
	AIChatFallbackMock      atomic.Int64
	AIChatError             atomic.Int64

	// OpenAI specific (attempted provider stats)
	AIEmbeddingSuccessOpenAI  atomic.Int64
	AIEmbeddingFallbackOpenAI atomic.Int64
	AIEmbeddingErrorOpenAI    atomic.Int64
	AIChatSuccessOpenAI       atomic.Int64
	AIChatFallbackOpenAI      atomic.Int64
	AIChatErrorOpenAI         atomic.Int64
)

// Snapshot returns a simple Prometheus-like exposition text (temporary helper).
func Snapshot() string {
	return fmt.Sprintf(`# AssistFusion metrics
assistfusion_ticket_created_total %d
assistfusion_ticket_assigned_total %d
assistfusion_ticket_escalated_total %d
assistfusion_ticket_resolved_total %d
assistfusion_ticket_reopened_total %d
assistfusion_kb_doc_created_total %d
assistfusion_kb_doc_updated_total %d
assistfusion_kb_doc_deleted_total %d
assistfusion_kb_search_requests_total %d
assistfusion_kb_search_hits_total %d
assistfusion_ai_embedding_calls_total %d
assistfusion_ai_embedding_success_mock_total %d
assistfusion_ai_embedding_fallback_mock_total %d
assistfusion_ai_embedding_error_total %d
assistfusion_ai_chat_success_mock_total %d
assistfusion_ai_chat_fallback_mock_total %d
assistfusion_ai_chat_error_total %d
assistfusion_ai_embedding_success_openai_total %d
assistfusion_ai_embedding_fallback_openai_total %d
assistfusion_ai_embedding_error_openai_total %d
assistfusion_ai_chat_success_openai_total %d
assistfusion_ai_chat_fallback_openai_total %d
assistfusion_ai_chat_error_openai_total %d

# pseudo-labeled series
assistfusion_ai_provider_calls_total{provider="mock",result="success"} %d
assistfusion_ai_provider_calls_total{provider="mock",result="fallback"} %d
assistfusion_ai_provider_calls_total{provider="mock",result="error"} %d
assistfusion_ai_provider_calls_total{provider="openai",result="success"} %d
assistfusion_ai_provider_calls_total{provider="openai",result="fallback"} %d
assistfusion_ai_provider_calls_total{provider="openai",result="error"} %d
`,
		TicketCreated.Load(),
		TicketAssigned.Load(),
		TicketEscalated.Load(),
		TicketResolved.Load(),
		TicketReopened.Load(),
		KBDocCreated.Load(),
		KBDocUpdated.Load(),
		KBDocDeleted.Load(),
		KBSearchRequests.Load(),
		KBSearchHits.Load(),
		AIEmbeddingCalls.Load(),
		AIEmbeddingSuccessMock.Load(),
		AIEmbeddingFallbackMock.Load(),
		AIEmbeddingError.Load(),
		AIChatSuccessMock.Load(),
		AIChatFallbackMock.Load(),
		AIChatError.Load(),
		AIEmbeddingSuccessOpenAI.Load(),
		AIEmbeddingFallbackOpenAI.Load(),
		AIEmbeddingErrorOpenAI.Load(),
		AIChatSuccessOpenAI.Load(),
		AIChatFallbackOpenAI.Load(),
		AIChatErrorOpenAI.Load(),
		// labeled aggregation lines (reuse counters)
		AIEmbeddingSuccessMock.Load(),
		AIEmbeddingFallbackMock.Load(),
		AIEmbeddingError.Load(),
		AIEmbeddingSuccessOpenAI.Load(),
		AIEmbeddingFallbackOpenAI.Load(),
		AIEmbeddingErrorOpenAI.Load(),
	)
}
