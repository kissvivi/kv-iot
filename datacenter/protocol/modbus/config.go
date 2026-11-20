package modbus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"kv-iot/datacenter/protocol"
)

const (
	defaultUnitID         = 1
	defaultConnectTimeout = 5 * time.Second
	defaultReadTimeout    = 3 * time.Second
	defaultWriteTimeout   = 3 * time.Second
)

// Config is the normalized Modbus TCP connection configuration.
type Config struct {
	Host           string
	Port           int
	UnitID         int
	AddressBase    int
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
}

type wireDuration struct {
	set   bool
	value time.Duration
	raw   string
}

func (d *wireDuration) UnmarshalJSON(data []byte) error {
	d.set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		value, err := time.ParseDuration(strings.TrimSpace(text))
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", text, err)
		}
		d.value = value
		d.raw = text
		return nil
	}

	var milliseconds json.Number
	if err := json.Unmarshal(data, &milliseconds); err != nil {
		return fmt.Errorf("duration must be milliseconds or a duration string: %w", err)
	}
	value, err := strconv.ParseInt(milliseconds.String(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid millisecond duration %q: %w", milliseconds.String(), err)
	}
	d.value = time.Duration(value) * time.Millisecond
	d.raw = milliseconds.String()
	return nil
}

type wireConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`

	UnitID      *int `json:"unitId"`
	UnitIDSnake *int `json:"unit_id"`

	AddressBase      *int `json:"addressBase"`
	AddressBaseSnake *int `json:"address_base"`

	ConnectTimeout      *wireDuration `json:"connectTimeout"`
	ConnectTimeoutSnake *wireDuration `json:"connect_timeout"`
	ConnectTimeoutMS    *wireDuration `json:"connectTimeoutMs"`
	ConnectTimeoutMSS   *wireDuration `json:"connect_timeout_ms"`

	ReadTimeout      *wireDuration `json:"readTimeout"`
	ReadTimeoutSnake *wireDuration `json:"read_timeout"`
	ReadTimeoutMS    *wireDuration `json:"readTimeoutMs"`
	ReadTimeoutMSS   *wireDuration `json:"read_timeout_ms"`

	WriteTimeout      *wireDuration `json:"writeTimeout"`
	WriteTimeoutSnake *wireDuration `json:"write_timeout"`
	WriteTimeoutMS    *wireDuration `json:"writeTimeoutMs"`
	WriteTimeoutMSS   *wireDuration `json:"write_timeout_ms"`

	Timeout      *wireDuration `json:"timeout"`
	TimeoutMS    *wireDuration `json:"timeoutMs"`
	TimeoutSnake *wireDuration `json:"timeout_ms"`
}

// ValidateConfig validates a Modbus TCP connection configuration.
func ValidateConfig(raw json.RawMessage) error {
	_, err := parseConfig(raw)
	return err
}

// Validate checks the normalized configuration.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("%w: host is required", protocol.ErrInvalidConfig)
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("%w: port must be between 1 and 65535", protocol.ErrInvalidConfig)
	}
	if c.UnitID < 0 || c.UnitID > 255 {
		return fmt.Errorf("%w: unitId must be between 0 and 255", protocol.ErrInvalidConfig)
	}
	if c.AddressBase != 0 && c.AddressBase != 1 {
		return fmt.Errorf("%w: addressBase must be 0 or 1", protocol.ErrInvalidConfig)
	}
	if c.ConnectTimeout <= 0 || c.ReadTimeout <= 0 || c.WriteTimeout <= 0 {
		return fmt.Errorf("%w: connect, read and write timeouts must be positive", protocol.ErrInvalidConfig)
	}
	return nil
}

func parseConfig(raw json.RawMessage) (Config, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return Config{}, fmt.Errorf("%w: config is required", protocol.ErrInvalidConfig)
	}

	var wire wireConfig
	if err := json.Unmarshal(raw, &wire); err != nil {
		return Config{}, fmt.Errorf("%w: decode modbus config: %v", protocol.ErrInvalidConfig, err)
	}

	cfg := Config{
		Host:           strings.TrimSpace(wire.Host),
		Port:           wire.Port,
		UnitID:         defaultUnitID,
		AddressBase:    0,
		ConnectTimeout: defaultConnectTimeout,
		ReadTimeout:    defaultReadTimeout,
		WriteTimeout:   defaultWriteTimeout,
	}

	unitID, set, err := pickInt("unitId", wire.UnitID, wire.UnitIDSnake)
	if err != nil {
		return Config{}, err
	}
	if set {
		cfg.UnitID = unitID
	}

	addressBase, set, err := pickInt("addressBase", wire.AddressBase, wire.AddressBaseSnake)
	if err != nil {
		return Config{}, err
	}
	if set {
		cfg.AddressBase = addressBase
	}

	commonTimeout, commonSet, err := pickDuration("timeout",
		wire.Timeout, wire.TimeoutMS, wire.TimeoutSnake)
	if err != nil {
		return Config{}, err
	}
	if commonSet {
		cfg.ConnectTimeout = commonTimeout
		cfg.ReadTimeout = commonTimeout
		cfg.WriteTimeout = commonTimeout
	}

	if value, set, err := pickDuration("connectTimeout",
		wire.ConnectTimeout, wire.ConnectTimeoutSnake,
		wire.ConnectTimeoutMS, wire.ConnectTimeoutMSS); err != nil {
		return Config{}, err
	} else if set {
		cfg.ConnectTimeout = value
	}
	if value, set, err := pickDuration("readTimeout",
		wire.ReadTimeout, wire.ReadTimeoutSnake,
		wire.ReadTimeoutMS, wire.ReadTimeoutMSS); err != nil {
		return Config{}, err
	} else if set {
		cfg.ReadTimeout = value
	}
	if value, set, err := pickDuration("writeTimeout",
		wire.WriteTimeout, wire.WriteTimeoutSnake,
		wire.WriteTimeoutMS, wire.WriteTimeoutMSS); err != nil {
		return Config{}, err
	} else if set {
		cfg.WriteTimeout = value
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func pickInt(name string, values ...*int) (int, bool, error) {
	var (
		result int
		found  bool
	)
	for _, value := range values {
		if value == nil {
			continue
		}
		if found && result != *value {
			return 0, false, fmt.Errorf("%w: conflicting %s fields", protocol.ErrInvalidConfig, name)
		}
		result = *value
		found = true
	}
	return result, found, nil
}

func pickDuration(name string, values ...*wireDuration) (time.Duration, bool, error) {
	var (
		result time.Duration
		found  bool
	)
	for _, value := range values {
		if value == nil || !value.set {
			continue
		}
		if found && result != value.value {
			return 0, false, fmt.Errorf("%w: conflicting %s fields", protocol.ErrInvalidConfig, name)
		}
		result = value.value
		found = true
	}
	return result, found, nil
}
