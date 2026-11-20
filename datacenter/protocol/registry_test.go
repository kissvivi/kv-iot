package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type fakeDriver struct {
	config DriverConfig
}

func (d *fakeDriver) Connect(ctx context.Context, config DriverConfig) error {
	d.config = config
	return nil
}

func (d *fakeDriver) Read(ctx context.Context, points []PointBinding) ([]PointSample, error) {
	return nil, nil
}

func (d *fakeDriver) Write(ctx context.Context, point PointBinding, value any) error {
	return nil
}

func (d *fakeDriver) Health() DriverHealth {
	return DriverHealth{Status: HealthHealthy}
}

func (d *fakeDriver) Close() error {
	return nil
}

func TestRegistryCreateAndValidate(t *testing.T) {
	registry := NewRegistry()
	var received DriverConfig
	validatorCalled := false

	err := registry.RegisterWithValidator(" Modbus_TCP ", func(config DriverConfig) (ProtocolDriver, error) {
		received = config
		return &fakeDriver{}, nil
	}, ConfigValidatorFunc(func(config json.RawMessage) error {
		validatorCalled = true
		if string(config) != `{"host":"127.0.0.1"}` {
			t.Fatalf("unexpected validator config: %s", config)
		}
		return nil
	}))
	if err != nil {
		t.Fatalf("register driver: %v", err)
	}

	config := DriverConfig{
		ChannelID: "channel-1",
		Config:    json.RawMessage(`{"host":"127.0.0.1"}`),
	}
	driver, err := registry.Create("modbus_tcp", config)
	if err != nil {
		t.Fatalf("create driver: %v", err)
	}
	if driver == nil {
		t.Fatal("expected driver")
	}
	if !validatorCalled {
		t.Fatal("expected protocol validator to be called")
	}
	if received.ProtocolType != ProtocolModbusTCP {
		t.Fatalf("expected protocol type to be populated, got %q", received.ProtocolType)
	}
	if got := registry.ProtocolTypes(); len(got) != 1 || got[0] != ProtocolModbusTCP {
		t.Fatalf("unexpected protocol types: %#v", got)
	}
}

func TestRegistryRejectsUnknownAndInvalidConfig(t *testing.T) {
	registry := NewRegistry()

	if _, err := registry.Create("mqtt", DriverConfig{}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected invalid config, got %v", err)
	}

	err := registry.Register("mqtt", func(config DriverConfig) (ProtocolDriver, error) {
		return &fakeDriver{}, nil
	})
	if err != nil {
		t.Fatalf("register driver: %v", err)
	}
	if err := registry.Validate("unknown", json.RawMessage(`{}`)); !errors.Is(err, ErrProtocolNotRegistered) {
		t.Fatalf("expected unknown protocol error, got %v", err)
	}
	if err := registry.Validate("mqtt", json.RawMessage(`{`)); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected invalid JSON error, got %v", err)
	}
}

func TestRegistryRejectsDuplicateDriver(t *testing.T) {
	registry := NewRegistry()
	factory := func(config DriverConfig) (ProtocolDriver, error) {
		return &fakeDriver{}, nil
	}
	if err := registry.Register("mqtt", factory); err != nil {
		t.Fatalf("register driver: %v", err)
	}
	if err := registry.Register("MQTT", factory); !errors.Is(err, ErrDriverAlreadyRegistered) {
		t.Fatalf("expected duplicate registration error, got %v", err)
	}
}
