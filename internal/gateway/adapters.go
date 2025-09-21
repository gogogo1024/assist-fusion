package gateway

import (
	"context"

	icommon "github.com/gogogo1024/assist-fusion/internal/common"
	grc "github.com/gogogo1024/assist-fusion/internal/gateway/rpc"
	"github.com/gogogo1024/assist-fusion/kitex_gen/ai/aiservice"
	kcommon "github.com/gogogo1024/assist-fusion/kitex_gen/common"
	"github.com/gogogo1024/assist-fusion/kitex_gen/kb"
	"github.com/gogogo1024/assist-fusion/kitex_gen/kb/kbservice"
	"github.com/gogogo1024/assist-fusion/kitex_gen/ticket"
	"github.com/gogogo1024/assist-fusion/kitex_gen/ticket/ticketservice"
)

// ----- Interfaces exposed to HTTP handlers -----

type TicketAPI interface {
	Create(ctx context.Context, title, desc, note string) (*kcommon.Ticket, error)
	Get(ctx context.Context, id string) (*kcommon.Ticket, error)
	List(ctx context.Context) ([]*kcommon.Ticket, error)
	Assign(ctx context.Context, id, note string) (*kcommon.Ticket, error)
	Resolve(ctx context.Context, id, note string) (*kcommon.Ticket, error)
	Escalate(ctx context.Context, id, note string) (*kcommon.Ticket, error)
	Reopen(ctx context.Context, id, note string) (*kcommon.Ticket, error)
	Cycles(ctx context.Context, id string) ([]*kcommon.TicketCycle, error)
	Events(ctx context.Context, id string) ([]*kcommon.TicketEvent, error)
}

type KBAPI interface {
	Add(ctx context.Context, title, content string) (*kcommon.KBDoc, error)
	Update(ctx context.Context, id, title, content string) (*kcommon.KBDoc, error)
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, q string, limit int32) ([]*kcommon.SearchItem, error)
	Info(ctx context.Context) (map[string]string, error)
}

type AIAPI interface {
	Embeddings(ctx context.Context, texts []string, dim int32) (*kcommon.EmbeddingResponse, error)
}

// ----- RPC implementations -----

// rpcClients removed: now using centralized gateway/rpc Init

// TicketAPI (RPC)
type ticketRPC struct{ c ticketservice.Client }

func (t *ticketRPC) Create(ctx context.Context, title, desc, note string) (*kcommon.Ticket, error) {
	req := &ticket.CreateTicketRequest{Title: title, Desc: desc}
	if note != "" {
		req.Note = &note
	}
	resp, err := t.c.CreateTicket(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetTicket(), nil
}
func (t *ticketRPC) Get(ctx context.Context, id string) (*kcommon.Ticket, error) {
	resp, err := t.c.GetTicket(ctx, &ticket.GetTicketRequest{Id: id})
	if err != nil {
		return nil, err
	}
	return resp.GetTicket(), nil
}
func (t *ticketRPC) List(ctx context.Context) ([]*kcommon.Ticket, error) {
	resp, err := t.c.ListTickets(ctx, &ticket.ListTicketsRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetTickets(), nil
}
func (t *ticketRPC) Assign(ctx context.Context, id, note string) (*kcommon.Ticket, error) {
	req := &ticket.TicketActionRequest{Id: id}
	if note != "" {
		req.Note = &note
	}
	resp, err := t.c.Assign(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetTicket(), nil
}
func (t *ticketRPC) Resolve(ctx context.Context, id, note string) (*kcommon.Ticket, error) {
	req := &ticket.TicketActionRequest{Id: id}
	if note != "" {
		req.Note = &note
	}
	resp, err := t.c.Resolve(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetTicket(), nil
}
func (t *ticketRPC) Escalate(ctx context.Context, id, note string) (*kcommon.Ticket, error) {
	req := &ticket.TicketActionRequest{Id: id}
	if note != "" {
		req.Note = &note
	}
	resp, err := t.c.Escalate(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetTicket(), nil
}
func (t *ticketRPC) Reopen(ctx context.Context, id, note string) (*kcommon.Ticket, error) {
	req := &ticket.TicketActionRequest{Id: id}
	if note != "" {
		req.Note = &note
	}
	resp, err := t.c.Reopen(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetTicket(), nil
}
func (t *ticketRPC) Cycles(ctx context.Context, id string) ([]*kcommon.TicketCycle, error) {
	return t.c.GetCycles(ctx, &ticket.GetCyclesRequest{Id: id})
}
func (t *ticketRPC) Events(ctx context.Context, id string) ([]*kcommon.TicketEvent, error) {
	return t.c.GetEvents(ctx, &ticket.GetEventsRequest{Id: id})
}

// KBAPI (RPC)
type kbRPC struct{ c kbservice.Client }

func (k *kbRPC) Add(ctx context.Context, title, content string) (*kcommon.KBDoc, error) {
	req := &kb.AddDocRequest{Title: title, Content: content}
	resp, err := k.c.AddDoc(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
func (k *kbRPC) Update(ctx context.Context, id, title, content string) (*kcommon.KBDoc, error) {
	req := &kb.UpdateDocRequest{Id: id}
	if title != "" {
		req.Title = &title
	}
	if content != "" {
		req.Content = &content
	}
	resp, err := k.c.UpdateDoc(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
func (k *kbRPC) Delete(ctx context.Context, id string) error {
	_, err := k.c.DeleteDoc(ctx, &kb.DeleteDocRequest{Id: id})
	return err
}
func (k *kbRPC) Search(ctx context.Context, q string, limit int32) ([]*kcommon.SearchItem, error) {
	req := &kb.SearchRequest{Query: q}
	if limit > 0 {
		req.Limit = &limit
	}
	resp, err := k.c.Search(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}
func (k *kbRPC) Info(ctx context.Context) (map[string]string, error) {
	resp, err := k.c.Info(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Stats, nil
}

// AIAPI (RPC)
type aiRPC struct{ c aiservice.Client }

func (a *aiRPC) Embeddings(ctx context.Context, texts []string, dim int32) (*kcommon.EmbeddingResponse, error) {
	return a.c.Embeddings(ctx, &kcommon.EmbeddingRequest{Texts: texts, Dim: dim})
}

// Factory to expose implementations
type RPCAdapter struct {
	Ticket TicketAPI
	KB     KBAPI
	AI     AIAPI
}

func NewRPCAdapter(cfg *icommon.Config) (*RPCAdapter, error) {
	if err := grc.Init(cfg); err != nil {
		return nil, err
	}
	return &RPCAdapter{Ticket: &ticketRPC{c: grc.TicketClient}, KB: &kbRPC{c: grc.KBClient}, AI: &aiRPC{c: grc.AIClient}}, nil
}

// end of adapters.go
