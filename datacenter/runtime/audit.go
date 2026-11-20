package runtime

import (
	"context"
	"sort"
	"sync"
	"time"
)

type AuditEvent struct {
	ID           string         `json:"id"`
	UserID       string         `json:"user_id,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	Detail       map[string]any `json:"detail,omitempty"`
	IP           string         `json:"ip,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type AuditSink interface {
	Record(ctx context.Context, event AuditEvent) error
}

type AuditReader interface {
	List(ctx context.Context, limit int) ([]AuditEvent, error)
}

type NoopAuditSink struct{}

func (NoopAuditSink) Record(ctx context.Context, event AuditEvent) error {
	return nil
}

type MemoryAuditStore struct {
	mu     sync.RWMutex
	events []AuditEvent
}

func NewMemoryAuditStore() *MemoryAuditStore {
	return &MemoryAuditStore{}
}

func (s *MemoryAuditStore) Record(ctx context.Context, event AuditEvent) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if event.ID == "" {
		event.ID = GenerateID("audit")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()
	return nil
}

func (s *MemoryAuditStore) List(ctx context.Context, limit int) ([]AuditEvent, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	events := append([]AuditEvent(nil), s.events...)
	s.mu.RUnlock()
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.After(events[j].CreatedAt)
	})
	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}
	return events, nil
}
