package state

import (
	"context"
	"errors"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
)

func TestMemoryStoreMarksExpiredPointStale(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStoreWithClock(func() time.Time { return now })
	sample := protocol.PointSample{
		DeviceID:        1,
		PointIdentifier: "temperature",
		Value:           25.5,
		Quality:         protocol.QualityGood,
		Timestamp:       now,
	}
	if err := store.Set(context.Background(), sample, time.Second); err != nil {
		t.Fatalf("set state: %v", err)
	}

	now = now.Add(2 * time.Second)
	got, err := store.Get(context.Background(), 1, "temperature")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if got.Quality != protocol.QualityStale {
		t.Fatalf("expected stale quality, got %q", got.Quality)
	}
}

func TestMemoryStoreHealthAndMissingPoint(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	if _, err := store.Get(ctx, 99, "missing"); !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	health := protocol.ChannelHealth{
		Status:      protocol.HealthHealthy,
		LastCheck:   time.Now(),
		LastSuccess: time.Now(),
	}
	if err := store.SetChannelHealth(ctx, "channel-1", health, time.Minute); err != nil {
		t.Fatalf("set health: %v", err)
	}
	got, err := store.GetChannelHealth(ctx, "channel-1")
	if err != nil {
		t.Fatalf("get health: %v", err)
	}
	if got.ChannelID != "channel-1" || got.Status != protocol.HealthHealthy {
		t.Fatalf("unexpected health: %#v", got)
	}
}
