package runtime

import (
	"context"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
	"kv-iot/datacenter/state"
)

func TestStateSampleSinkWritesLatestState(t *testing.T) {
	store := state.NewMemoryStore()
	sink, err := NewStateSampleSink(store, StateSampleSinkOptions{
		DefaultTTL: time.Minute,
	})
	if err != nil {
		t.Fatalf("new state sink: %v", err)
	}
	sample := protocol.PointSample{
		ChannelID:       "channel-1",
		DeviceID:        7,
		PointIdentifier: "temperature",
		Value:           21.5,
		Quality:         protocol.QualityGood,
		Timestamp:       time.Now(),
	}
	if err := sink.Publish(context.Background(), []protocol.PointSample{sample}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	stored, err := store.Get(context.Background(), sample.DeviceID, sample.PointIdentifier)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if stored.Value != sample.Value {
		t.Fatalf("unexpected value: %#v", stored.Value)
	}
}
