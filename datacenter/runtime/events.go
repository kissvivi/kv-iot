package runtime

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type Event struct {
	ID              string         `json:"id"`
	ChannelID       string         `json:"channel_id,omitempty"`
	DeviceID        int64          `json:"device_id,omitempty"`
	DeviceNo        string         `json:"device_no,omitempty"`
	EventIdentifier string         `json:"event_identifier"`
	Severity        string         `json:"severity,omitempty"`
	Payload         map[string]any `json:"payload,omitempty"`
	OccurredAt      time.Time      `json:"occurred_at"`
}

type EventFilter struct {
	ChannelID string
	DeviceID  int64
	Severity  string
	Since     time.Time
	Limit     int
}

type EventStore interface {
	Append(ctx context.Context, event Event) error
	List(ctx context.Context, filter EventFilter) ([]Event, error)
}

type MemoryEventStore struct {
	mu     sync.RWMutex
	events []Event
}

func NewMemoryEventStore() *MemoryEventStore {
	return &MemoryEventStore{}
}

func (s *MemoryEventStore) Append(ctx context.Context, event Event) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if event.ID == "" {
		event.ID = GenerateID("event")
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now()
	}
	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()
	return nil
}

func (s *MemoryEventStore) List(ctx context.Context, filter EventFilter) ([]Event, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	s.mu.RLock()
	events := make([]Event, 0, len(s.events))
	for _, event := range s.events {
		if filter.ChannelID != "" && event.ChannelID != filter.ChannelID {
			continue
		}
		if filter.DeviceID != 0 && event.DeviceID != filter.DeviceID {
			continue
		}
		if filter.Severity != "" && !strings.EqualFold(event.Severity, filter.Severity) {
			continue
		}
		if !filter.Since.IsZero() && event.OccurredAt.Before(filter.Since) {
			continue
		}
		events = append(events, event)
	}
	s.mu.RUnlock()

	sort.Slice(events, func(i, j int) bool {
		return events[i].OccurredAt.After(events[j].OccurredAt)
	})
	if len(events) > limit {
		events = events[:limit]
	}
	return events, nil
}
