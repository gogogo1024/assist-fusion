package impl

import (
	"context"
	"time"

	"github.com/gogogo1024/assist-fusion/internal/common"
	"github.com/gogogo1024/assist-fusion/internal/observability"
	kcommon "github.com/gogogo1024/assist-fusion/kitex_gen/common"
	"github.com/gogogo1024/assist-fusion/kitex_gen/ticket"
	"github.com/google/uuid"
)

type TicketServiceImpl struct{ Repo common.TicketRepo }

func NewTicketService(repo common.TicketRepo) *TicketServiceImpl {
	return &TicketServiceImpl{Repo: repo}
}

const (
	notFoundMsg   = "not found"
	idRequiredMsg = "id required"
)

func toThriftTicket(t *common.Ticket) *kcommon.Ticket {
	if t == nil {
		return nil
	}
	statusEnum := kcommon.TicketStatus_CREATED
	switch t.Status {
	case "assigned":
		statusEnum = kcommon.TicketStatus_ASSIGNED
	case "escalated":
		statusEnum = kcommon.TicketStatus_ESCALATED
	case "resolved":
		statusEnum = kcommon.TicketStatus_RESOLVED
	}
	cycles := make([]*kcommon.TicketCycle, 0, len(t.Cycles))
	for _, c := range t.Cycles {
		sc := kcommon.TicketStatus_CREATED
		switch c.Status {
		case "assigned":
			sc = kcommon.TicketStatus_ASSIGNED
		case "escalated":
			sc = kcommon.TicketStatus_ESCALATED
		case "resolved":
			sc = kcommon.TicketStatus_RESOLVED
		}
		cycles = append(cycles, &kcommon.TicketCycle{CreatedAt: c.CreatedAt, AssignedAt: c.AssignedAt, ResolvedAt: c.ResolvedAt, EscalatedAt: c.EscalatedAt, Status: sc})
	}
	events := make([]*kcommon.TicketEvent, 0, len(t.Events))
	for _, e := range t.Events {
		events = append(events, &kcommon.TicketEvent{Type: e.Type, At: e.At, Note: e.Note})
	}
	return &kcommon.Ticket{Id: t.ID, Title: t.Title, Desc: t.Desc, Status: statusEnum, CreatedAt: t.CreatedAt, AssignedAt: t.AssignedAt, ResolvedAt: t.ResolvedAt, EscalatedAt: t.EscalatedAt, ReopenedAt: t.ReopenedAt, Cycles: cycles, CurrentCycle: int32(t.CurrentCycle), Events: events}
}

func (s *TicketServiceImpl) CreateTicket(ctx context.Context, req *ticket.CreateTicketRequest) (*ticket.TicketResponse, error) {
	if req == nil || req.Title == "" {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeBadRequest, Message: "title required"}
	}
	note := ""
	if req.Note != nil {
		note = *req.Note
	}
	now := time.Now().Unix()
	t := &common.Ticket{ID: uuid.NewString(), Title: req.Title, Desc: req.Desc, Status: "created", CreatedAt: now, Cycles: []common.TicketCycle{{CreatedAt: now, Status: "created"}}, CurrentCycle: 0, Events: []common.TicketEvent{{Type: "created", At: now, Note: note}}}
	_ = s.Repo.Create(ctx, t)
	observability.TicketCreated.Add(1)
	return &ticket.TicketResponse{Ticket: toThriftTicket(t)}, nil
}
func (s *TicketServiceImpl) GetTicket(ctx context.Context, req *ticket.GetTicketRequest) (*ticket.TicketResponse, error) {
	if req == nil || req.Id == "" {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeBadRequest, Message: idRequiredMsg}
	}
	t, _ := s.Repo.Get(ctx, req.Id)
	if t == nil {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeNotFound, Message: notFoundMsg}
	}
	return &ticket.TicketResponse{Ticket: toThriftTicket(t)}, nil
}
func (s *TicketServiceImpl) ListTickets(ctx context.Context, req *ticket.ListTicketsRequest) (*ticket.ListTicketsResponse, error) {
	ts, _ := s.Repo.List(ctx)
	out := make([]*kcommon.Ticket, 0, len(ts))
	for _, t := range ts {
		out = append(out, toThriftTicket(t))
	}
	return &ticket.ListTicketsResponse{Tickets: out}, nil
}
func (s *TicketServiceImpl) Assign(ctx context.Context, req *ticket.TicketActionRequest) (*ticket.TicketResponse, error) {
	if req == nil || req.Id == "" {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeBadRequest, Message: idRequiredMsg}
	}
	t, _ := s.Repo.Get(ctx, req.Id)
	if t == nil {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeNotFound, Message: notFoundMsg}
	}
	note := ""
	if req.Note != nil {
		note = *req.Note
	}
	now := time.Now().Unix()
	t.AssignedAt = now
	t.Status = "assigned"
	if t.CurrentCycle >= 0 && t.CurrentCycle < len(t.Cycles) {
		cyc := &t.Cycles[t.CurrentCycle]
		cyc.AssignedAt = now
		cyc.Status = "assigned"
	}
	t.Events = append(t.Events, common.TicketEvent{Type: "assigned", At: now, Note: note})
	_ = s.Repo.Update(ctx, t)
	observability.TicketAssigned.Add(1)
	return &ticket.TicketResponse{Ticket: toThriftTicket(t)}, nil
}
func (s *TicketServiceImpl) Resolve(ctx context.Context, req *ticket.TicketActionRequest) (*ticket.TicketResponse, error) {
	if req == nil || req.Id == "" {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeBadRequest, Message: idRequiredMsg}
	}
	t, _ := s.Repo.Get(ctx, req.Id)
	if t == nil {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeNotFound, Message: notFoundMsg}
	}
	note := ""
	if req.Note != nil {
		note = *req.Note
	}
	now := time.Now().Unix()
	t.ResolvedAt = now
	t.Status = "resolved"
	if t.CurrentCycle >= 0 && t.CurrentCycle < len(t.Cycles) {
		cyc := &t.Cycles[t.CurrentCycle]
		cyc.ResolvedAt = now
		cyc.Status = "resolved"
	}
	t.Events = append(t.Events, common.TicketEvent{Type: "resolved", At: now, Note: note})
	_ = s.Repo.Update(ctx, t)
	observability.TicketResolved.Add(1)
	return &ticket.TicketResponse{Ticket: toThriftTicket(t)}, nil
}
func (s *TicketServiceImpl) Escalate(ctx context.Context, req *ticket.TicketActionRequest) (*ticket.TicketResponse, error) {
	if req == nil || req.Id == "" {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeBadRequest, Message: idRequiredMsg}
	}
	t, _ := s.Repo.Get(ctx, req.Id)
	if t == nil {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeNotFound, Message: notFoundMsg}
	}
	note := ""
	if req.Note != nil {
		note = *req.Note
	}
	now := time.Now().Unix()
	t.EscalatedAt = now
	t.Status = "escalated"
	if t.CurrentCycle >= 0 && t.CurrentCycle < len(t.Cycles) {
		cyc := &t.Cycles[t.CurrentCycle]
		cyc.EscalatedAt = now
		cyc.Status = "escalated"
	}
	t.Events = append(t.Events, common.TicketEvent{Type: "escalated", At: now, Note: note})
	_ = s.Repo.Update(ctx, t)
	observability.TicketEscalated.Add(1)
	return &ticket.TicketResponse{Ticket: toThriftTicket(t)}, nil
}
func (s *TicketServiceImpl) Reopen(ctx context.Context, req *ticket.TicketActionRequest) (*ticket.TicketResponse, error) {
	if req == nil || req.Id == "" {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeBadRequest, Message: idRequiredMsg}
	}
	t, _ := s.Repo.Get(ctx, req.Id)
	if t == nil {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeNotFound, Message: notFoundMsg}
	}
	note := ""
	if req.Note != nil {
		note = *req.Note
	}
	now := time.Now().Unix()
	t.ReopenedAt = now
	t.Cycles = append(t.Cycles, common.TicketCycle{CreatedAt: now, Status: "reopened"})
	t.CurrentCycle = len(t.Cycles) - 1
	t.Events = append(t.Events, common.TicketEvent{Type: "reopened", At: now, Note: note})
	_ = s.Repo.Update(ctx, t)
	observability.TicketReopened.Add(1)
	return &ticket.TicketResponse{Ticket: toThriftTicket(t)}, nil
}
func (s *TicketServiceImpl) GetCycles(ctx context.Context, req *ticket.GetCyclesRequest) ([]*kcommon.TicketCycle, error) {
	if req == nil || req.Id == "" {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeBadRequest, Message: idRequiredMsg}
	}
	t, _ := s.Repo.Get(ctx, req.Id)
	if t == nil {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeNotFound, Message: notFoundMsg}
	}
	out := make([]*kcommon.TicketCycle, 0, len(t.Cycles))
	for _, c := range t.Cycles {
		sc := kcommon.TicketStatus_CREATED
		switch c.Status {
		case "assigned":
			sc = kcommon.TicketStatus_ASSIGNED
		case "escalated":
			sc = kcommon.TicketStatus_ESCALATED
		case "resolved":
			sc = kcommon.TicketStatus_RESOLVED
		}
		out = append(out, &kcommon.TicketCycle{CreatedAt: c.CreatedAt, AssignedAt: c.AssignedAt, ResolvedAt: c.ResolvedAt, EscalatedAt: c.EscalatedAt, Status: sc})
	}
	return out, nil
}
func (s *TicketServiceImpl) GetEvents(ctx context.Context, req *ticket.GetEventsRequest) ([]*kcommon.TicketEvent, error) {
	if req == nil || req.Id == "" {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeBadRequest, Message: idRequiredMsg}
	}
	t, _ := s.Repo.Get(ctx, req.Id)
	if t == nil {
		return nil, &kcommon.ServiceError{Code: common.ErrCodeNotFound, Message: notFoundMsg}
	}
	out := make([]*kcommon.TicketEvent, 0, len(t.Events))
	for _, e := range t.Events {
		out = append(out, &kcommon.TicketEvent{Type: e.Type, At: e.At, Note: e.Note})
	}
	return out, nil
}
