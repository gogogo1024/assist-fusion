package router

// Path constants centralizing HTTP routes.
const (
	PathTickets        = "/v1/tickets"
	PathTicketID       = "/v1/tickets/:id"
	PathTicketAssign   = "/v1/tickets/:id/assign"
	PathTicketResolve  = "/v1/tickets/:id/resolve"
	PathTicketEscalate = "/v1/tickets/:id/escalate"
	PathTicketStart    = "/v1/tickets/:id/start"
	PathTicketWait     = "/v1/tickets/:id/wait"
	PathTicketClose    = "/v1/tickets/:id/close"
	PathTicketCancel   = "/v1/tickets/:id/cancel"
	PathTicketReopen   = "/v1/tickets/:id/reopen"
	PathTicketCycles   = "/v1/tickets/:id/cycles"
	PathTicketEvents   = "/v1/tickets/:id/events"

	PathDocs       = "/v1/docs"
	PathDocID      = "/v1/docs/:id"
	PathSearch     = "/v1/search"
	PathKBInfo     = "/v1/kb/info"
	PathEmbeddings = "/v1/embeddings"
)
