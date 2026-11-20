package state

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"kv-iot/datacenter/protocol"
)

type memoryPointEntry struct {
	sample    protocol.PointSample
	expiresAt time.Time
}

type memoryHealthEntry struct {
	health    protocol.ChannelHealth
	expiresAt time.Time
}

// MemoryStore is a process-local StateStore for tests, development and
// fallback operation. Expired points remain readable with stale quality.
type MemoryStore struct {
	mu     sync.RWMutex
	points map[string]memoryPointEntry
	health map[string]memoryHealthEntry
	now    func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return NewMemoryStoreWithClock(time.Now)
}

func NewMemoryStoreWithClock(now func() time.Time) *MemoryStore {
	if now == nil {
		now = time.Now
	}
	return &MemoryStore{
		points: make(map[string]memoryPointEntry),
		health: make(map[string]memoryHealthEntry),
		now:    now,
	}
}

func (s *MemoryStore) Set(ctx context.Context, sample protocol.PointSample, ttl time.Duration) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(sample.PointIdentifier) == "" {
		return fmt.Errorf("point identifier is required")
	}
	now := s.now()
	s.mu.Lock()
	s.points[memoryPointKey(sample.DeviceID, sample.PointIdentifier)] = memoryPointEntry{
		sample:    cloneSample(sample),
		expiresAt: expiration(now, ttl),
	}
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, deviceID int64, pointIdentifier string) (*protocol.PointSample, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	entry, exists := s.points[memoryPointKey(deviceID, pointIdentifier)]
	s.mu.RUnlock()
	if !exists {
		return nil, ErrStateNotFound
	}

	sample := cloneSample(entry.sample)
	if expired(s.now(), entry.expiresAt) {
		sample.Quality = protocol.QualityStale
	}
	return &sample, nil
}

func (s *MemoryStore) ListByDevice(ctx context.Context, deviceID int64) ([]protocol.PointSample, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	now := s.now()
	prefix := memoryDevicePrefix(deviceID)

	s.mu.RLock()
	samples := make([]protocol.PointSample, 0)
	for key, entry := range s.points {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		sample := cloneSample(entry.sample)
		if expired(now, entry.expiresAt) {
			sample.Quality = protocol.QualityStale
		}
		samples = append(samples, sample)
	}
	s.mu.RUnlock()

	sort.Slice(samples, func(i, j int) bool {
		return samples[i].PointIdentifier < samples[j].PointIdentifier
	})
	return samples, nil
}

func (s *MemoryStore) Delete(ctx context.Context, deviceID int64, pointIdentifier string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.points, memoryPointKey(deviceID, pointIdentifier))
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) SetChannelHealth(ctx context.Context, channelID string, health protocol.ChannelHealth, ttl time.Duration) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(channelID) == "" {
		return fmt.Errorf("channel id is required")
	}
	health.ChannelID = channelID
	s.mu.Lock()
	s.health[channelID] = memoryHealthEntry{
		health:    health,
		expiresAt: expiration(s.now(), ttl),
	}
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) GetChannelHealth(ctx context.Context, channelID string) (*protocol.ChannelHealth, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	entry, exists := s.health[channelID]
	s.mu.RUnlock()
	if !exists || expired(s.now(), entry.expiresAt) {
		return nil, ErrStateNotFound
	}
	health := entry.health
	return &health, nil
}

func memoryPointKey(deviceID int64, pointIdentifier string) string {
	return memoryDevicePrefix(deviceID) + pointIdentifier
}

func memoryDevicePrefix(deviceID int64) string {
	return fmt.Sprintf("device:%d:", deviceID)
}

func expiration(now time.Time, ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return now.Add(ttl)
}

func expired(now, expiresAt time.Time) bool {
	return !expiresAt.IsZero() && !now.Before(expiresAt)
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func cloneSample(sample protocol.PointSample) protocol.PointSample {
	return sample
}
