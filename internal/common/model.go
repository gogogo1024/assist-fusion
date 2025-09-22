package common

import (
	"context"
	"errors"
)

// Minimal subset recreated after cleanup to support ticket RPC & probe.
// Error code constants referenced by RPC implementations.
const (
	ErrCodeBadRequest    = "bad_request"
	ErrCodeNotFound      = "not_found"
	ErrCodeConflict      = "conflict"
	ErrCodeKBUnavailable = "kb_unavailable"
	ErrCodeInternal      = "internal_error"
)

// Ticket domain model (simplified) kept for in-memory probe & RPC service.
type Ticket struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Desc         string        `json:"desc"`
	Status       string        `json:"status"`
	CreatedAt    int64         `json:"created_at"`
	AssignedAt   int64         `json:"assigned_at"`
	ResolvedAt   int64         `json:"resolved_at"`
	EscalatedAt  int64         `json:"escalated_at"`
	ReopenedAt   int64         `json:"reopened_at"`
	ClosedAt     int64         `json:"closed_at"`
	CanceledAt   int64         `json:"canceled_at"`
	Assignee     string        `json:"assignee"`
	Priority     string        `json:"priority"`
	Customer     string        `json:"customer"`
	Category     string        `json:"category"`
	Tags         []string      `json:"tags"`
	DueAt        int64         `json:"due_at"`
	Cycles       []TicketCycle `json:"cycles,omitempty"`
	CurrentCycle int           `json:"current_cycle"`
	Events       []TicketEvent `json:"events,omitempty"`
}

// TicketCycle stores timestamps of one lifecycle iteration.
type TicketCycle struct {
	CreatedAt   int64  `json:"created_at"`
	AssignedAt  int64  `json:"assigned_at"`
	ResolvedAt  int64  `json:"resolved_at"`
	EscalatedAt int64  `json:"escalated_at"`
	ClosedAt    int64  `json:"closed_at"`
	CanceledAt  int64  `json:"canceled_at"`
	Status      string `json:"status"`
}

// TicketEvent is an immutable audit entry.
type TicketEvent struct {
	Type string `json:"type"`
	At   int64  `json:"at"`
	Note string `json:"note"`
}

// TicketRepo defines required persistence operations.
type TicketRepo interface {
	Create(ctx context.Context, t *Ticket) error
	Get(ctx context.Context, id string) (*Ticket, error)
	List(ctx context.Context) ([]*Ticket, error)
	Update(ctx context.Context, t *Ticket) error
	Delete(ctx context.Context, id string) error
}

// MemoryTicketRepo simple in-memory implementation (non-concurrent) retained for ticket RPC & probe.
type MemoryTicketRepo struct{ store map[string]*Ticket }

func NewMemoryTicketRepo() *MemoryTicketRepo {
	return &MemoryTicketRepo{store: make(map[string]*Ticket)}
}

func (r *MemoryTicketRepo) Create(ctx context.Context, t *Ticket) error {
	r.store[t.ID] = t
	return nil
}
func (r *MemoryTicketRepo) Get(ctx context.Context, id string) (*Ticket, error) {
	if t, ok := r.store[id]; ok {
		return t, nil
	}
	return nil, nil
}
func (r *MemoryTicketRepo) List(ctx context.Context) ([]*Ticket, error) {
	out := make([]*Ticket, 0, len(r.store))
	for _, t := range r.store {
		out = append(out, t)
	}
	return out, nil
}
func (r *MemoryTicketRepo) Update(ctx context.Context, t *Ticket) error {
	if _, ok := r.store[t.ID]; !ok {
		return ErrNotFound
	}
	r.store[t.ID] = t
	return nil
}
func (r *MemoryTicketRepo) Delete(ctx context.Context, id string) error {
	delete(r.store, id)
	return nil
}

// ErrNotFound sentinel for missing ticket in repo.
var ErrNotFound = errors.New("not found")
