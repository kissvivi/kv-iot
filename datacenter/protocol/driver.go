package protocol

import "context"

// DriverFactory creates a driver instance. Connection is established later by
// the channel worker so the factory remains cheap and side-effect free.
type DriverFactory func(config DriverConfig) (ProtocolDriver, error)

// ProtocolDriver is the common point-based protocol contract. MQTT, Modbus
// TCP, TCP frame and future protocols implement this interface.
type ProtocolDriver interface {
	Connect(ctx context.Context, config DriverConfig) error
	Read(ctx context.Context, points []PointBinding) ([]PointSample, error)
	Write(ctx context.Context, point PointBinding, value any) error
	Health() DriverHealth
	Close() error
}

// StreamDriver is implemented by drivers that receive unsolicited payloads,
// such as MQTT uplinks or a persistent TCP byte stream. Subscribe must return
// when ctx is canceled.
type StreamDriver interface {
	Subscribe(ctx context.Context, handler func([]byte)) error
}

// RawDriver exposes a request/response byte transport for TCP passthrough.
type RawDriver interface {
	SendRaw(ctx context.Context, payload []byte) ([]byte, error)
}

// RawStreamDriver is a capability interface for drivers that support both raw
// request/response and unsolicited streams.
type RawStreamDriver interface {
	StreamDriver
	RawDriver
}
