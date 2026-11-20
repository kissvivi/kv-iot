package storage

import (
	"context"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
)

func TestMemoryHistoryStoreFiltersAndTrims(t *testing.T) {
	store := NewMemoryHistoryStore(3)
	now := time.Now()
	samples := []protocol.PointSample{
		{DeviceID: 1, PointIdentifier: "temperature", Value: 1, Timestamp: now.Add(-3 * time.Minute)},
		{DeviceID: 1, PointIdentifier: "temperature", Value: 2, Timestamp: now.Add(-2 * time.Minute)},
		{DeviceID: 2, PointIdentifier: "temperature", Value: 3, Timestamp: now.Add(-time.Minute)},
		{DeviceID: 1, PointIdentifier: "humidity", Value: 4, Timestamp: now},
	}
	if err := store.WritePointSamples(context.Background(), samples); err != nil {
		t.Fatalf("write samples: %v", err)
	}
	got, err := store.QueryPointSamples(context.Background(), TelemetryQuery{
		DeviceID:        1,
		PointIdentifier: "temperature",
		From:            now.Add(-3 * time.Minute),
		To:              now,
		Limit:           10,
	})
	if err != nil {
		t.Fatalf("query samples: %v", err)
	}
	if len(got) != 1 || got[0].Value != 2 {
		t.Fatalf("filtered samples = %#v, want one sample with value 2", got)
	}

	got, err = store.QueryPointSamples(context.Background(), TelemetryQuery{
		DeviceID: 1,
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("query limited samples: %v", err)
	}
	if len(got) != 1 || got[0].PointIdentifier != "humidity" {
		t.Fatalf("limited samples = %#v, want latest humidity sample", got)
	}
}
