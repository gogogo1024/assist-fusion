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
	)
}
