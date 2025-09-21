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
	ID           string
	Title        string
	Desc         string
	Status       string
	CreatedAt    int64
	AssignedAt   int64
	ResolvedAt   int64
	EscalatedAt  int64
	ReopenedAt   int64
	ClosedAt     int64
	CanceledAt   int64
	Assignee     string
	Priority     string
	Customer     string
	Category     string
	Tags         []string
	DueAt        int64
	Cycles       []TicketCycle
	CurrentCycle int
	Events       []TicketEvent
}

// TicketCycle stores timestamps of one lifecycle iteration.
type TicketCycle struct {
	CreatedAt   int64
	AssignedAt  int64
	ResolvedAt  int64
	EscalatedAt int64
	ClosedAt    int64
	CanceledAt  int64
	Status      string
}

// TicketEvent is an immutable audit entry.
type TicketEvent struct {
	Type string
	At   int64
	Note string
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
