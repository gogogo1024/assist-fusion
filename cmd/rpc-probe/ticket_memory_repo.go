package main

// Extracted minimal in-memory ticket repository from deprecated internal/common/store.go
// to keep rpc-probe self-contained.

import (
	"context"
	"errors"
)

// Ticket represents a simplified ticket model used only by the probe.
type Ticket struct {
	ID          string
	Title       string
	Desc        string
	Status      string
	CreatedAt   int64
	AssignedAt  int64
	ResolvedAt  int64
	EscalatedAt int64
	ReopenedAt  int64
	ClosedAt    int64
	CanceledAt  int64
	Assignee    string
	Priority    string
	Customer    string
	Category    string
	Tags        []string
	DueAt       int64
}

// TicketRepo defines required operations for inline probe servers.
type TicketRepo interface {
	Create(ctx context.Context, t *Ticket) error
	Get(ctx context.Context, id string) (*Ticket, error)
	List(ctx context.Context) ([]*Ticket, error)
	Update(ctx context.Context, t *Ticket) error
	Delete(ctx context.Context, id string) error
}

// MemoryTicketRepo is an in-memory implementation (not concurrency safe; fine for probe usage).
type MemoryTicketRepo struct {
	store map[string]*Ticket
}

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

var errNotFound = errors.New("not found")

func (r *MemoryTicketRepo) Update(ctx context.Context, t *Ticket) error {
	if _, ok := r.store[t.ID]; !ok {
		return errNotFound
	}
	r.store[t.ID] = t
	return nil
}
func (r *MemoryTicketRepo) Delete(ctx context.Context, id string) error {
	delete(r.store, id)
	return nil
}
