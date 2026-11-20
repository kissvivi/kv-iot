package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
	modbusdriver "kv-iot/datacenter/protocol/modbus"
	"kv-iot/datacenter/runtime"
	"kv-iot/datacenter/state"
	"kv-iot/internal/testsupport/modbusserver"
)

// TODO(QA-INT-02): Exercise the runtime HTTP API through cmd/datacenter once
// the API and Manager are wired into the service entrypoint.
// TODO(QA-INT-02): Add TCP raw/frame Manager integration once channel
// provisioning and runtime API coverage are available.
func TestRuntimeManagerStateSinkAndEndpointProtection(t *testing.T) {
	server := modbusserver.New(modbusserver.Config{HoldingRegisters: 8})
	if err := server.Start(); err != nil {
		t.Fatalf("start Modbus simulator: %v", err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("close Modbus simulator: %v", err)
		}
	})
	if err := server.Data().SetHoldingRegister(0, 42); err != nil {
		t.Fatalf("seed holding register: %v", err)
	}

	host, port := splitHostPort(t, server.Addr())
	connectionConfig := mustJSON(t, map[string]any{
		"host":         host,
		"port":         port,
		"unitId":       1,
		"readTimeout":  "1s",
		"writeTimeout": "1s",
	})

	registry := protocol.NewRegistry()
	if err := registry.RegisterWithValidator(
		protocol.ProtocolModbusTCP,
		modbusdriver.NewDriver,
		protocol.ConfigValidatorFunc(modbusdriver.ValidateConfig),
	); err != nil {
		t.Fatalf("register Modbus driver: %v", err)
	}

	store := state.NewMemoryStore()
	stateSink, err := runtime.NewStateSampleSink(store, runtime.StateSampleSinkOptions{
		DefaultTTL: time.Minute,
	})
	if err != nil {
		t.Fatalf("create state sink: %v", err)
	}
	published := make(chan []protocol.PointSample, 4)
	sink := runtime.NewSinkGroup(
		stateSink,
		runtime.SampleSinkFunc(func(_ context.Context, samples []protocol.PointSample) error {
			copied := append([]protocol.PointSample(nil), samples...)
			select {
			case published <- copied:
			default:
			}
			return nil
		}),
	)

	manager, err := runtime.NewManager(registry, sink, runtime.WorkerOptions{
		PollInterval:      20 * time.Millisecond,
		HealthInterval:    20 * time.Millisecond,
		ReconnectInterval: time.Second,
		ConnectTimeout:    time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      time.Second,
	})
	if err != nil {
		t.Fatalf("create runtime manager: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pollPoint := protocol.PointBinding{
		PointID:         301,
		ProductID:       9,
		ProductKey:      "p9",
		DeviceID:        301,
		DeviceNo:        "device-301",
		PointIdentifier: "counter",
		DataType:        "uint16",
		Config: mustJSON(t, map[string]any{
			"area":     "holding_register",
			"address":  0,
			"dataType": "uint16",
		}),
	}
	firstSpec := runtime.ChannelSpec{
		ChannelID:    "runtime-channel-a",
		InstanceID:   41,
		ProtocolType: protocol.ProtocolModbusTCP,
		EndpointKey:  "modbus-simulator-endpoint",
		Config:       connectionConfig,
		PollInterval: 20 * time.Millisecond,
		PollPoints:   []protocol.PointBinding{pollPoint},
	}
	if err := manager.StartChannel(ctx, firstSpec); err != nil {
		t.Fatalf("start first runtime channel: %v", err)
	}
	t.Cleanup(func() {
		if err := manager.StopAll(context.Background()); err != nil {
			t.Errorf("stop runtime channels: %v", err)
		}
	})

	select {
	case batch := <-published:
		if len(batch) != 1 || batch[0].PointIdentifier != "counter" {
			t.Fatalf("unexpected published batch: %#v", batch)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for runtime sample sink")
	}

	waitFor(t, 3*time.Second, "state store sample", func() bool {
		sample, getErr := store.Get(context.Background(), 301, "counter")
		return getErr == nil && sample.Value == uint16(42)
	})
	snapshot, exists := manager.Snapshot(firstSpec.ChannelID)
	if !exists {
		t.Fatalf("runtime channel %q not found", firstSpec.ChannelID)
	}
	if snapshot.Status != runtime.ChannelStatusRunning {
		t.Fatalf("runtime channel status = %s, want running", snapshot.Status)
	}

	if err := manager.StartChannel(ctx, firstSpec); !errors.Is(err, runtime.ErrChannelAlreadyRunning) {
		t.Fatalf("duplicate channel error = %v, want ErrChannelAlreadyRunning", err)
	}

	secondSpec := firstSpec
	secondSpec.ChannelID = "runtime-channel-b"
	secondSpec.InstanceID = 42
	if err := manager.StartChannel(ctx, secondSpec); !errors.Is(err, runtime.ErrEndpointAlreadyRunning) {
		t.Fatalf("duplicate endpoint error = %v, want ErrEndpointAlreadyRunning", err)
	}

	if err := manager.StopChannel(ctx, firstSpec.ChannelID); err != nil {
		t.Fatalf("stop first runtime channel: %v", err)
	}
	if _, exists := manager.Get(firstSpec.ChannelID); exists {
		t.Fatalf("stopped runtime channel %q is still registered", firstSpec.ChannelID)
	}
	if err := manager.StartChannel(ctx, secondSpec); err != nil {
		t.Fatalf("start channel after endpoint release: %v", err)
	}
	if snapshot, exists := manager.Snapshot(secondSpec.ChannelID); !exists ||
		snapshot.Status != runtime.ChannelStatusRunning {
		t.Fatalf("replacement runtime channel snapshot = %#v, exists=%v", snapshot, exists)
	}
}
