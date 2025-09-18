package gateway

import (
	"context"

	"github.com/cloudwego/kitex/client"
	"github.com/gogogo1024/assist-fusion/internal/common"
	"github.com/gogogo1024/assist-fusion/kitex_gen/ai/aiservice"
	kcommon "github.com/gogogo1024/assist-fusion/kitex_gen/common"
	"github.com/gogogo1024/assist-fusion/kitex_gen/kb/kbservice"
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

type rpcClients struct {
	ticket ticketservice.Client
	kb     kbservice.Client
	ai     aiservice.Client
}

func NewRPCClients(cfg *common.Config) (*rpcClients, error) {
	tCli, err := ticketservice.NewClient("ticket-rpc", client.WithHostPorts(cfg.TicketRPCAddr))
	if err != nil {
		return nil, err
	}
	kCli, err := kbservice.NewClient("kb-rpc", client.WithHostPorts(cfg.KBRPCAddr))
	if err != nil {
		return nil, err
	}
	aCli, err := aiservice.NewClient("ai-rpc", client.WithHostPorts(cfg.AIRPCAddr))
	if err != nil {
		return nil, err
	}
	return &rpcClients{ticket: tCli, kb: kCli, ai: aCli}, nil
}

// TicketAPI (RPC)
type ticketRPC struct{ c ticketservice.Client }

func (t *ticketRPC) Create(ctx context.Context, title, desc, note string) (*kcommon.Ticket, error) {
	return t.c.CreateTicket(ctx, title, desc, note)
}
func (t *ticketRPC) Get(ctx context.Context, id string) (*kcommon.Ticket, error) {
	return t.c.GetTicket(ctx, id)
}
func (t *ticketRPC) List(ctx context.Context) ([]*kcommon.Ticket, error) { return t.c.ListTickets(ctx) }
func (t *ticketRPC) Assign(ctx context.Context, id, note string) (*kcommon.Ticket, error) {
	return t.c.Assign(ctx, id, note)
}
func (t *ticketRPC) Resolve(ctx context.Context, id, note string) (*kcommon.Ticket, error) {
	return t.c.Resolve(ctx, id, note)
}
func (t *ticketRPC) Escalate(ctx context.Context, id, note string) (*kcommon.Ticket, error) {
	return t.c.Escalate(ctx, id, note)
}
func (t *ticketRPC) Reopen(ctx context.Context, id, note string) (*kcommon.Ticket, error) {
	return t.c.Reopen(ctx, id, note)
}
func (t *ticketRPC) Cycles(ctx context.Context, id string) ([]*kcommon.TicketCycle, error) {
	return t.c.GetCycles(ctx, id)
}
func (t *ticketRPC) Events(ctx context.Context, id string) ([]*kcommon.TicketEvent, error) {
	return t.c.GetEvents(ctx, id)
}

// KBAPI (RPC)
type kbRPC struct{ c kbservice.Client }

func (k *kbRPC) Add(ctx context.Context, title, content string) (*kcommon.KBDoc, error) {
	return k.c.AddDoc(ctx, title, content)
}
func (k *kbRPC) Update(ctx context.Context, id, title, content string) (*kcommon.KBDoc, error) {
	return k.c.UpdateDoc(ctx, id, title, content)
}
func (k *kbRPC) Delete(ctx context.Context, id string) error { return k.c.DeleteDoc(ctx, id) }
func (k *kbRPC) Search(ctx context.Context, q string, limit int32) ([]*kcommon.SearchItem, error) {
	return k.c.Search(ctx, q, limit)
}
func (k *kbRPC) Info(ctx context.Context) (map[string]string, error) { return k.c.Info(ctx) }

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

func NewRPCAdapter(cfg *common.Config) (*RPCAdapter, error) {
	cs, err := NewRPCClients(cfg)
	if err != nil {
		return nil, err
	}
	return &RPCAdapter{Ticket: &ticketRPC{c: cs.ticket}, KB: &kbRPC{c: cs.kb}, AI: &aiRPC{c: cs.ai}}, nil
}

// end of adapters.go
