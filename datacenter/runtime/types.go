package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kv-iot/datacenter/protocol"
)

type ChannelStatus string

const (
	ChannelStatusDisabled   ChannelStatus = "disabled"
	ChannelStatusConnecting ChannelStatus = "connecting"
	ChannelStatusRunning    ChannelStatus = "running"
	ChannelStatusDegraded   ChannelStatus = "degraded"
	ChannelStatusError      ChannelStatus = "error"
	ChannelStatusStopped    ChannelStatus = "stopped"
)

// ChannelSpec is independent of the device model so tests and future
// configuration providers can start workers without importing device data.
type ChannelSpec struct {
	ChannelID          string                   `json:"channel_id"`
	InstanceID         int64                    `json:"instance_id,omitempty"`
	Name               string                   `json:"name,omitempty"`
	ProtocolType       string                   `json:"protocol_type"`
	DriverName         string                   `json:"driver_name,omitempty"`
	EndpointKey        string                   `json:"endpoint_key,omitempty"`
	Config             json.RawMessage          `json:"config"`
	PollInterval       time.Duration            `json:"poll_interval,omitempty"`
	PollGroupIntervals map[string]time.Duration `json:"poll_group_intervals,omitempty"`
	PollPoints         []protocol.PointBinding  `json:"poll_points,omitempty"`
	Enabled            bool                     `json:"enabled"`
	Metadata           map[string]string        `json:"metadata,omitempty"`
}

func (s ChannelSpec) Validate() error {
	if strings.TrimSpace(s.ChannelID) == "" {
		return fmt.Errorf("%w: channel_id is required", ErrInvalidChannelSpec)
	}
	if strings.TrimSpace(s.ProtocolType) == "" {
		return fmt.Errorf("%w: protocol_type is required", ErrInvalidChannelSpec)
	}
	if len(strings.TrimSpace(string(s.Config))) == 0 {
		return fmt.Errorf("%w: config is required", ErrInvalidChannelSpec)
	}
	if !json.Valid(s.Config) {
		return fmt.Errorf("%w: config is not valid JSON", ErrInvalidChannelSpec)
	}
	for index, point := range s.PollPoints {
		if err := point.Validate(); err != nil {
			return fmt.Errorf("%w: poll point %d: %v", ErrInvalidChannelSpec, index, err)
		}
	}
	for name, interval := range s.PollGroupIntervals {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("%w: poll group name is required", ErrInvalidChannelSpec)
		}
		if interval <= 0 {
			return fmt.Errorf("%w: poll group %s interval must be positive", ErrInvalidChannelSpec, name)
		}
	}
	return nil
}

func (s ChannelSpec) DriverConfig() protocol.DriverConfig {
	return protocol.DriverConfig{
		ChannelID:    s.ChannelID,
		ProtocolType: s.ProtocolType,
		DriverName:   s.DriverName,
		Config:       append(json.RawMessage(nil), s.Config...),
	}
}

func (s ChannelSpec) RuntimeEndpointKey() string {
	if key := strings.TrimSpace(s.EndpointKey); key != "" {
		return key
	}
	return strings.TrimSpace(s.ChannelID)
}

// SampleSink receives normalized samples from a worker. Implementations
// usually fan out to Redis, InfluxDB and a live event stream.
type SampleSink interface {
	Publish(ctx context.Context, samples []protocol.PointSample) error
}

type SampleSinkFunc func(ctx context.Context, samples []protocol.PointSample) error

func (f SampleSinkFunc) Publish(ctx context.Context, samples []protocol.PointSample) error {
	if f == nil {
		return nil
	}
	return f(ctx, samples)
}

type NullSink struct{}

func (NullSink) Publish(ctx context.Context, samples []protocol.PointSample) error {
	return nil
}

type ChannelSnapshot struct {
	ChannelID    string                 `json:"channel_id"`
	InstanceID   int64                  `json:"instance_id,omitempty"`
	Name         string                 `json:"name,omitempty"`
	ProtocolType string                 `json:"protocol_type"`
	EndpointKey  string                 `json:"endpoint_key"`
	Status       ChannelStatus          `json:"status"`
	Health       protocol.ChannelHealth `json:"health"`
	PointCount   int                    `json:"point_count"`
	PollGroups   []PollGroupSnapshot    `json:"poll_groups,omitempty"`
	StartedAt    time.Time              `json:"started_at,omitempty"`
	LastReadAt   time.Time              `json:"last_read_at,omitempty"`
	LastSampleAt time.Time              `json:"last_sample_at,omitempty"`
	LastWriteAt  time.Time              `json:"last_write_at,omitempty"`
	LastErrorAt  time.Time              `json:"last_error_at,omitempty"`
	LastError    string                 `json:"last_error,omitempty"`
}

type PollGroupSnapshot struct {
	Name       string `json:"name"`
	IntervalMs int64  `json:"interval_ms"`
	PointCount int    `json:"point_count"`
}

type WorkerOptions struct {
	PollInterval      time.Duration
	HealthInterval    time.Duration
	ReconnectInterval time.Duration
	ConnectTimeout    time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
}

func (o WorkerOptions) withDefaults() WorkerOptions {
	if o.PollInterval <= 0 {
		o.PollInterval = 5 * time.Second
	}
	if o.HealthInterval <= 0 {
		o.HealthInterval = 10 * time.Second
	}
	if o.ReconnectInterval <= 0 {
		o.ReconnectInterval = 5 * time.Second
	}
	if o.ConnectTimeout <= 0 {
		o.ConnectTimeout = 5 * time.Second
	}
	if o.ReadTimeout <= 0 {
		o.ReadTimeout = 3 * time.Second
	}
	if o.WriteTimeout <= 0 {
		o.WriteTimeout = 3 * time.Second
	}
	return o
}
