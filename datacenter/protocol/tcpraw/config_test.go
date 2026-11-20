package tcpraw

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
)

func TestParseConfig(t *testing.T) {
	raw := json.RawMessage(`{
		"host": "127.0.0.1",
		"port": 9000,
		"timeout_ms": 750,
		"reconnect_delay": "100ms",
		"max_reconnect_attempts": 2,
		"response_mode": "delimiter",
		"responseDelimiterHex": "0d0a"
	}`)

	cfg, err := parseConfig(raw)
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if cfg.ConnectTimeout != 750*time.Millisecond {
		t.Fatalf("connect timeout = %s", cfg.ConnectTimeout)
	}
	if cfg.ResponseMode != ResponseModeDelimiter {
		t.Fatalf("response mode = %q", cfg.ResponseMode)
	}
	if string(cfg.ResponseDelimiter) != "\r\n" {
		t.Fatalf("delimiter = %q", cfg.ResponseDelimiter)
	}
	if cfg.MaxReconnectAttempts != 2 {
		t.Fatalf("reconnect attempts = %d", cfg.MaxReconnectAttempts)
	}
}

func TestValidateConfigRejectsInvalidResponseMode(t *testing.T) {
	raw := json.RawMessage(`{"host":"127.0.0.1","port":502,"responseMode":"packet"}`)
	err := ValidateConfig(raw)
	if !errors.Is(err, protocol.ErrInvalidConfig) {
		t.Fatalf("error = %v, want ErrInvalidConfig", err)
	}
}

func TestValidateConfigRequiresDelimiter(t *testing.T) {
	raw := json.RawMessage(`{"host":"127.0.0.1","port":502,"responseMode":"delimiter"}`)
	err := ValidateConfig(raw)
	if !errors.Is(err, protocol.ErrInvalidConfig) {
		t.Fatalf("error = %v, want ErrInvalidConfig", err)
	}
}
