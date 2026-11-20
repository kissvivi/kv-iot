package storage

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"kv-iot/datacenter/protocol"
)

type TelemetryQuery struct {
	DeviceID        int64
	PointIdentifier string
	From            time.Time
	To              time.Time
	Limit           int
}

type SeriesReader interface {
	QueryPointSamples(ctx context.Context, query TelemetryQuery) ([]protocol.PointSample, error)
}

type FallbackReader struct {
	Primary   SeriesReader
	Secondary SeriesReader
}

func (r FallbackReader) QueryPointSamples(ctx context.Context, query TelemetryQuery) ([]protocol.PointSample, error) {
	if r.Primary != nil {
		samples, err := r.Primary.QueryPointSamples(ctx, query)
		if err == nil && len(samples) > 0 {
			return samples, nil
		}
	}
	if r.Secondary != nil {
		return r.Secondary.QueryPointSamples(ctx, query)
	}
	return nil, nil
}

type MemoryHistoryStore struct {
	mu      sync.RWMutex
	max     int
	samples []protocol.PointSample
}

var (
	_ SeriesWriter = (*MemoryHistoryStore)(nil)
	_ SeriesReader = (*MemoryHistoryStore)(nil)
)

func NewMemoryHistoryStore(max int) *MemoryHistoryStore {
	if max <= 0 {
		max = 10000
	}
	return &MemoryHistoryStore{
		max:     max,
		samples: make([]protocol.PointSample, 0, max),
	}
}

func (s *MemoryHistoryStore) WritePointSamples(ctx context.Context, samples []protocol.PointSample) error {
	if len(samples) == 0 {
		return nil
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.samples = append(s.samples, samples...)
	if overflow := len(s.samples) - s.max; overflow > 0 {
		copy(s.samples, s.samples[overflow:])
		s.samples = s.samples[:len(s.samples)-overflow]
	}
	return nil
}

func (s *MemoryHistoryStore) Publish(ctx context.Context, samples []protocol.PointSample) error {
	return s.WritePointSamples(ctx, samples)
}

func (s *MemoryHistoryStore) QueryPointSamples(ctx context.Context, query TelemetryQuery) ([]protocol.PointSample, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]protocol.PointSample, 0)
	for _, sample := range s.samples {
		if query.DeviceID > 0 && sample.DeviceID != query.DeviceID {
			continue
		}
		if identifier := strings.TrimSpace(query.PointIdentifier); identifier != "" &&
			sample.PointIdentifier != identifier {
			continue
		}
		if !query.From.IsZero() && sample.Timestamp.Before(query.From) {
			continue
		}
		if !query.To.IsZero() && sample.Timestamp.After(query.To) {
			continue
		}
		result = append(result, sample)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})
	if query.Limit > 0 && len(result) > query.Limit {
		result = result[len(result)-query.Limit:]
	}
	return result, nil
}
