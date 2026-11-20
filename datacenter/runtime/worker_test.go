package runtime

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
)

type testDriver struct {
	mu        sync.Mutex
	connected bool
	reads     int
	writes    int
}

func (d *testDriver) Connect(ctx context.Context, config protocol.DriverConfig) error {
	d.mu.Lock()
	d.connected = true
	d.mu.Unlock()
	return nil
}

func (d *testDriver) Read(ctx context.Context, points []protocol.PointBinding) ([]protocol.PointSample, error) {
	d.mu.Lock()
	d.reads++
	d.mu.Unlock()
	return []protocol.PointSample{{
		PointIdentifier: points[0].PointIdentifier,
		Value:           1,
		Quality:         protocol.QualityGood,
	}}, nil
}

func (d *testDriver) Write(ctx context.Context, point protocol.PointBinding, value any) error {
	d.mu.Lock()
	d.writes++
	d.mu.Unlock()
	return nil
}

func (d *testDriver) Health() protocol.DriverHealth {
	return protocol.DriverHealth{Status: protocol.HealthHealthy}
}

func (d *testDriver) Close() error {
	d.mu.Lock()
	d.connected = false
	d.mu.Unlock()
	return nil
}

func TestWorkerPollsWritesAndStops(t *testing.T) {
	driver := &testDriver{}
	samples := make(chan []protocol.PointSample, 4)
	sink := SampleSinkFunc(func(ctx context.Context, points []protocol.PointSample) error {
		samples <- points
		return nil
	})
	spec := ChannelSpec{
		ChannelID:    "channel-1",
		ProtocolType: protocol.ProtocolModbusTCP,
		Config:       json.RawMessage(`{}`),
		PollInterval: 10 * time.Millisecond,
		PollPoints: []protocol.PointBinding{{
			PointIdentifier: "temperature",
			Config:          json.RawMessage(`{"address":1}`),
		}},
	}
	worker, err := NewWorker(spec, driver, sink, WorkerOptions{
		HealthInterval: 20 * time.Millisecond,
		ReadTimeout:    time.Second,
		WriteTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}
	if err := worker.Start(context.Background()); err != nil {
		t.Fatalf("start worker: %v", err)
	}

	select {
	case batch := <-samples:
		if len(batch) != 1 || batch[0].ChannelID != spec.ChannelID {
			t.Fatalf("unexpected sample batch: %#v", batch)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for samples")
	}

	if err := worker.Write(context.Background(), spec.PollPoints[0], 2); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := worker.Stop(context.Background()); err != nil {
		t.Fatalf("stop worker: %v", err)
	}
	if snapshot := worker.Snapshot(); snapshot.Status != ChannelStatusStopped {
		t.Fatalf("expected stopped status, got %s", snapshot.Status)
	}
}

func TestWorkerBuildsPollGroupsWithConfiguredIntervals(t *testing.T) {
	spec := ChannelSpec{
		ChannelID:    "channel-groups",
		ProtocolType: protocol.ProtocolModbusTCP,
		Config:       json.RawMessage(`{}`),
		PollInterval: 5 * time.Second,
		PollGroupIntervals: map[string]time.Duration{
			"fast": time.Second,
		},
		PollPoints: []protocol.PointBinding{
			{PointIdentifier: "fast_value", PollGroup: "fast"},
			{PointIdentifier: "slow_value", PollGroup: "slow"},
			{PointIdentifier: "default_value"},
		},
	}
	worker, err := NewWorker(spec, &testDriver{}, NullSink{}, WorkerOptions{})
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}
	groups := worker.pollGroups()
	if len(groups) != 3 {
		t.Fatalf("poll groups = %#v, want 3 groups", groups)
	}
	if groups["fast"].interval != time.Second {
		t.Fatalf("fast interval = %s, want 1s", groups["fast"].interval)
	}
	if groups["slow"].interval != 5*time.Second || groups["default"].interval != 5*time.Second {
		t.Fatalf("default intervals = %#v", groups)
	}
	snapshot := worker.Snapshot()
	if len(snapshot.PollGroups) != 3 || snapshot.PollGroups[0].Name != "default" {
		t.Fatalf("poll group snapshot = %#v", snapshot.PollGroups)
	}
}

func TestManagerRejectsDuplicateEndpoint(t *testing.T) {
	registry := protocol.NewRegistry()
	if err := registry.Register(protocol.ProtocolModbusTCP, func(config protocol.DriverConfig) (protocol.ProtocolDriver, error) {
		return &testDriver{}, nil
	}); err != nil {
		t.Fatalf("register driver: %v", err)
	}
	manager, err := NewManager(registry, NullSink{}, WorkerOptions{})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	first := ChannelSpec{
		ChannelID:    "channel-1",
		EndpointKey:  "gateway-1",
		ProtocolType: protocol.ProtocolModbusTCP,
		Config:       json.RawMessage(`{}`),
	}
	if err := manager.StartChannel(context.Background(), first); err != nil {
		t.Fatalf("start first channel: %v", err)
	}
	defer manager.StopAll(context.Background())

	second := first
	second.ChannelID = "channel-2"
	if err := manager.StartChannel(context.Background(), second); err == nil {
		t.Fatal("expected duplicate endpoint to be rejected")
	}
}
