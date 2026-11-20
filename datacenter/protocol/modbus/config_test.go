package modbus

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
)

func TestParseConfigDefaultsAndAliases(t *testing.T) {
	raw := json.RawMessage(`{
		"host": " 127.0.0.1 ",
		"port": 502,
		"unit_id": 7,
		"address_base": 1,
		"connect_timeout_ms": 250,
		"readTimeout": "2s",
		"write_timeout": 3000
	}`)

	cfg, err := parseConfig(raw)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Host != "127.0.0.1" || cfg.Port != 502 {
		t.Fatalf("unexpected endpoint: %#v", cfg)
	}
	if cfg.UnitID != 7 || cfg.AddressBase != 1 {
		t.Fatalf("unexpected addressing config: %#v", cfg)
	}
	if cfg.ConnectTimeout != 250*time.Millisecond {
		t.Fatalf("connect timeout = %s", cfg.ConnectTimeout)
	}
	if cfg.ReadTimeout != 2*time.Second || cfg.WriteTimeout != 3*time.Second {
		t.Fatalf("unexpected timeouts: %#v", cfg)
	}
}

func TestParseConfigCommonTimeoutAndDefaults(t *testing.T) {
	cfg, err := parseConfig(json.RawMessage(`{
		"host": "device.local",
		"port": 1502,
		"timeout_ms": 750
	}`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.UnitID != defaultUnitID || cfg.AddressBase != 0 {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
	if cfg.ConnectTimeout != 750*time.Millisecond ||
		cfg.ReadTimeout != 750*time.Millisecond ||
		cfg.WriteTimeout != 750*time.Millisecond {
		t.Fatalf("common timeout was not applied: %#v", cfg)
	}
}

func TestValidateConfigRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "missing host", raw: `{"port":502}`},
		{name: "invalid port", raw: `{"host":"127.0.0.1","port":70000}`},
		{name: "invalid unit", raw: `{"host":"127.0.0.1","port":502,"unitId":256}`},
		{name: "invalid address base", raw: `{"host":"127.0.0.1","port":502,"addressBase":2}`},
		{name: "negative timeout", raw: `{"host":"127.0.0.1","port":502,"readTimeoutMs":-1}`},
		{name: "zero timeout", raw: `{"host":"127.0.0.1","port":502,"timeout":"0s"}`},
		{name: "conflicting unit", raw: `{"host":"127.0.0.1","port":502,"unitId":1,"unit_id":2}`},
		{name: "malformed json", raw: `{"host":`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateConfig(json.RawMessage(test.raw))
			if err == nil {
				t.Fatal("expected configuration error")
			}
			if !errors.Is(err, protocol.ErrInvalidConfig) {
				t.Fatalf("error = %v, want ErrInvalidConfig", err)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	cfg := Config{
		Host:           "127.0.0.1",
		Port:           502,
		UnitID:         1,
		AddressBase:    0,
		ConnectTimeout: time.Second,
		ReadTimeout:    time.Second,
		WriteTimeout:   time.Second,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config: %v", err)
	}
}
