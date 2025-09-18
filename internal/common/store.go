package common

import (
	"context"
	"errors"
)

// Ticket model

// ...existing code...
type Ticket struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Desc        string `json:"desc"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	AssignedAt  int64  `json:"assigned_at"`
	ResolvedAt  int64  `json:"resolved_at"`
	EscalatedAt int64  `json:"escalated_at"`
	ReopenedAt  int64  `json:"reopened_at"`
	// Cycles-based modeling (B): each reopen creates a new cycle
	Cycles       []TicketCycle `json:"cycles"`
	CurrentCycle int           `json:"current_cycle"`
	// Events capture an immutable audit trail of ticket transitions
	Events []TicketEvent `json:"events"`
}

// TicketCycle represents one handling round for a ticket
type TicketCycle struct {
	CreatedAt   int64  `json:"created_at"`
	AssignedAt  int64  `json:"assigned_at"`
	ResolvedAt  int64  `json:"resolved_at"`
	EscalatedAt int64  `json:"escalated_at"`
	Status      string `json:"status"`
}

// TicketEvent is an audit entry of a ticket transition or action
type TicketEvent struct {
	Type string `json:"type"`
	At   int64  `json:"at"`
	Note string `json:"note"`
}

// TicketRepo interface

type TicketRepo interface {
	Create(ctx context.Context, t *Ticket) error
	Get(ctx context.Context, id string) (*Ticket, error)
	List(ctx context.Context) ([]*Ticket, error)
	Update(ctx context.Context, t *Ticket) error
	Delete(ctx context.Context, id string) error
}

// MemoryTicketRepo implements TicketRepo in memory

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
	t, ok := r.store[id]
	if !ok {
		return nil, nil
	}
	return t, nil
}
func (r *MemoryTicketRepo) List(ctx context.Context) ([]*Ticket, error) {
	var out []*Ticket
	for _, t := range r.store {
		out = append(out, t)
	}
	return out, nil
}

var ErrNotFound = errors.New("not found")

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
