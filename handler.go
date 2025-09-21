//go:build ignore

// Ignored legacy stub file. Real implementations live under services/*-rpc.
package main

import (
	ai "github.com/gogogo1024/assist-fusion/kitex_gen/ai"
	common "github.com/gogogo1024/assist-fusion/kitex_gen/common"
	kb "github.com/gogogo1024/assist-fusion/kitex_gen/kb"
	ticket "github.com/gogogo1024/assist-fusion/kitex_gen/ticket"
)

// CreateTicket implements the TicketServiceImpl interface.
func (s *TicketServiceImpl) CreateTicket(ctx context.Context, req *ticket.CreateTicketRequest) (resp *ticket.TicketResponse, err error) {
	// TODO: Your code here...
	return
}

// GetTicket implements the TicketServiceImpl interface.
func (s *TicketServiceImpl) GetTicket(ctx context.Context, req *ticket.GetTicketRequest) (resp *ticket.TicketResponse, err error) {
	// TODO: Your code here...
	return
}

// ListTickets implements the TicketServiceImpl interface.
func (s *TicketServiceImpl) ListTickets(ctx context.Context, req *ticket.ListTicketsRequest) (resp *ticket.ListTicketsResponse, err error) {
	// TODO: Your code here...
	return
}

// Assign implements the TicketServiceImpl interface.
func (s *TicketServiceImpl) Assign(ctx context.Context, req *ticket.TicketActionRequest) (resp *ticket.TicketResponse, err error) {
	// TODO: Your code here...
	return
}

// Resolve implements the TicketServiceImpl interface.
func (s *TicketServiceImpl) Resolve(ctx context.Context, req *ticket.TicketActionRequest) (resp *ticket.TicketResponse, err error) {
	// TODO: Your code here...
	return
}

// Escalate implements the TicketServiceImpl interface.
func (s *TicketServiceImpl) Escalate(ctx context.Context, req *ticket.TicketActionRequest) (resp *ticket.TicketResponse, err error) {
	// TODO: Your code here...
	return
}

// Reopen implements the TicketServiceImpl interface.
func (s *TicketServiceImpl) Reopen(ctx context.Context, req *ticket.TicketActionRequest) (resp *ticket.TicketResponse, err error) {
	// TODO: Your code here...
	return
}

// GetCycles implements the TicketServiceImpl interface.
func (s *TicketServiceImpl) GetCycles(ctx context.Context, req *ticket.GetCyclesRequest) (resp []*common.TicketCycle, err error) {
	// TODO: Your code here...
	return
}

// GetEvents implements the TicketServiceImpl interface.
func (s *TicketServiceImpl) GetEvents(ctx context.Context, req *ticket.GetEventsRequest) (resp []*common.TicketEvent, err error) {
	// TODO: Your code here...
	return
}

// AddDoc implements the KBServiceImpl interface.
func (s *KBServiceImpl) AddDoc(ctx context.Context, req *kb.AddDocRequest) (resp *common.KBDoc, err error) {
	// TODO: Your code here...
	return
}

// UpdateDoc implements the KBServiceImpl interface.
func (s *KBServiceImpl) UpdateDoc(ctx context.Context, req *kb.UpdateDocRequest) (resp *common.KBDoc, err error) {
	// TODO: Your code here...
	return
}

// DeleteDoc implements the KBServiceImpl interface.
func (s *KBServiceImpl) DeleteDoc(ctx context.Context, req *kb.DeleteDocRequest) (resp *kb.DeleteDocResponse, err error) {
	// TODO: Your code here...
	return
}

// Search implements the KBServiceImpl interface.
func (s *KBServiceImpl) Search(ctx context.Context, req *kb.SearchRequest) (resp *kb.SearchResponse, err error) {
	// TODO: Your code here...
	return
}

// Info implements the KBServiceImpl interface.
func (s *KBServiceImpl) Info(ctx context.Context) (resp *kb.InfoResponse, err error) {
	// TODO: Your code here...
	return
}

// Embeddings implements the AIServiceImpl interface.
func (s *AIServiceImpl) Embeddings(ctx context.Context, req *common.EmbeddingRequest) (resp *common.EmbeddingResponse, err error) {
	// TODO: Your code here...
	return
}

// Chat implements the AIServiceImpl interface.
func (s *AIServiceImpl) Chat(ctx context.Context, req *ai.ChatRequest) (resp *ai.ChatResponse, err error) {
	// TODO: Your code here...
	return
}
