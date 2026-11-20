package protocol

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	ProtocolMQTT      = "mqtt"
	ProtocolModbusTCP = "modbus_tcp"
	ProtocolTCPRaw    = "tcp_raw"
	ProtocolTCPFrame  = "tcp_frame"
	ProtocolS7        = "s7"
)

const (
	QualityGood          = "good"
	QualityStale         = "stale"
	QualityBad           = "bad"
	QualityTimeout       = "timeout"
	QualityProtocolError = "protocol_error"
)

// HealthStatus is the normalized connection health shared by drivers,
// workers and the state store.
type HealthStatus string

const (
	HealthUnknown    HealthStatus = "unknown"
	HealthConnecting HealthStatus = "connecting"
	HealthHealthy    HealthStatus = "healthy"
	HealthDegraded   HealthStatus = "degraded"
	HealthError      HealthStatus = "error"
	HealthStopped    HealthStatus = "stopped"
)

// DriverConfig is the transport-neutral configuration passed to a driver.
// Driver-specific settings remain as JSON so new protocols do not require
// changes to the runtime contract.
type DriverConfig struct {
	ChannelID    string          `json:"channel_id"`
	ProtocolType string          `json:"protocol_type,omitempty"`
	DriverName   string          `json:"driver_name,omitempty"`
	Config       json.RawMessage `json:"config"`
}

// Validate checks fields shared by every driver. A driver validator handles
// protocol-specific fields.
func (c DriverConfig) Validate() error {
	if strings.TrimSpace(c.ChannelID) == "" {
		return fmt.Errorf("%w: channel_id is required", ErrInvalidConfig)
	}
	if strings.TrimSpace(c.ProtocolType) == "" {
		return fmt.Errorf("%w: protocol_type is required", ErrInvalidConfig)
	}
	if len(strings.TrimSpace(string(c.Config))) == 0 {
		return fmt.Errorf("%w: config is required", ErrInvalidConfig)
	}
	if !json.Valid(c.Config) {
		return fmt.Errorf("%w: config is not valid JSON", ErrInvalidConfig)
	}
	return nil
}

// PointBinding is the protocol-neutral input for a point read or write.
// Protocol-specific addressing is carried in Config.
type PointBinding struct {
	PointID         int64           `json:"point_id"`
	ProductID       int64           `json:"product_id"`
	ProductKey      string          `json:"product_key,omitempty"`
	DeviceID        int64           `json:"device_id"`
	DeviceNo        string          `json:"device_no,omitempty"`
	PointIdentifier string          `json:"point_identifier"`
	PointKind       string          `json:"point_kind,omitempty"`
	CollectionMode  string          `json:"collection_mode,omitempty"`
	DataType        string          `json:"data_type,omitempty"`
	PollGroup       string          `json:"poll_group,omitempty"`
	ReadOnly        bool            `json:"read_only,omitempty"`
	Config          json.RawMessage `json:"config,omitempty"`
}

// Validate checks metadata required to correlate a sample with a point.
func (p PointBinding) Validate() error {
	if strings.TrimSpace(p.PointIdentifier) == "" {
		return fmt.Errorf("%w: point_identifier is required", ErrInvalidConfig)
	}
	if len(p.Config) > 0 && !json.Valid(p.Config) {
		return fmt.Errorf("%w: point config is not valid JSON", ErrInvalidConfig)
	}
	return nil
}

// PointSample is the normalized output of every protocol driver.
type PointSample struct {
	ChannelID       string    `json:"channel_id"`
	ProductID       int64     `json:"product_id"`
	ProductKey      string    `json:"product_key"`
	DeviceID        int64     `json:"device_id"`
	DeviceNo        string    `json:"device_no"`
	PointID         int64     `json:"point_id"`
	PointIdentifier string    `json:"point_identifier"`
	PointKind       string    `json:"point_kind"`
	DataType        string    `json:"data_type"`
	Value           any       `json:"value"`
	Quality         string    `json:"quality"`
	Timestamp       time.Time `json:"timestamp"`
	Sequence        uint64    `json:"sequence"`
}

// ChannelHealth is shared by runtime status queries and Redis state.
type ChannelHealth struct {
	ChannelID           string       `json:"channel_id"`
	InstanceID          int64        `json:"instance_id,omitempty"`
	Status              HealthStatus `json:"status"`
	Message             string       `json:"message,omitempty"`
	Since               time.Time    `json:"since"`
	LastCheck           time.Time    `json:"last_check"`
	LastSuccess         time.Time    `json:"last_success"`
	LastErrorAt         time.Time    `json:"last_error_at,omitempty"`
	LastError           string       `json:"last_error,omitempty"`
	ConsecutiveFailures int          `json:"consecutive_failures"`
}

type DriverHealth = ChannelHealth

// Healthy reports whether the driver can currently serve requests.
func (h ChannelHealth) Healthy() bool {
	return h.Status == HealthHealthy
}

// ConfigValidator validates a protocol-specific JSON configuration without
// creating a live driver.
type ConfigValidator interface {
	ValidateConfig(config json.RawMessage) error
}

type ConfigValidatorFunc func(config json.RawMessage) error

func (f ConfigValidatorFunc) ValidateConfig(config json.RawMessage) error {
	if f == nil {
		return nil
	}
	return f(config)
}
