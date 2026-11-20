package tcpframe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"kv-iot/datacenter/protocol"
)

// Driver implements one persistent TCP frame connection. All network
// operations are serialized because request/response protocols share one
// stream and cannot safely interleave frames.
type Driver struct {
	connectMu sync.Mutex
	opMu      sync.Mutex
	mu        sync.RWMutex

	conn      net.Conn
	config    Config
	channelID string
	closed    bool
	health    protocol.ChannelHealth
}

var _ protocol.ProtocolDriver = (*Driver)(nil)

// New creates a TCP frame driver without opening a connection.
func New(config protocol.DriverConfig) (*Driver, error) {
	if config.ChannelID != "" {
		if err := config.Validate(); err != nil {
			return nil, err
		}
	}
	cfg, err := parseConfig(config.Config)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &Driver{
		config:    cfg,
		channelID: config.ChannelID,
		health: protocol.ChannelHealth{
			ChannelID: config.ChannelID,
			Status:    protocol.HealthUnknown,
			Since:     now,
			LastCheck: now,
		},
	}, nil
}

// NewDriver is the DriverFactory-compatible constructor.
func NewDriver(config protocol.DriverConfig) (protocol.ProtocolDriver, error) {
	return New(config)
}

// Connect opens the configured TCP connection.
func (d *Driver) Connect(ctx context.Context, config protocol.DriverConfig) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if config.ChannelID != "" {
		if err := config.Validate(); err != nil {
			return err
		}
	}
	cfg, err := parseConfig(config.Config)
	if err != nil {
		return err
	}
	return d.connect(ctx, config.ChannelID, cfg, true)
}

func (d *Driver) connect(ctx context.Context, channelID string, cfg Config, explicit bool) error {
	d.connectMu.Lock()
	defer d.connectMu.Unlock()

	now := time.Now()
	d.mu.Lock()
	if explicit {
		d.closed = false
	} else if d.closed {
		d.mu.Unlock()
		return ErrClosed
	}
	oldConn := d.conn
	d.conn = nil
	d.config = cfg
	d.channelID = channelID
	d.health = protocol.ChannelHealth{
		ChannelID: channelID,
		Status:    protocol.HealthConnecting,
		Message:   "connecting",
		Since:     now,
		LastCheck: now,
	}
	d.mu.Unlock()
	if oldConn != nil {
		_ = oldConn.Close()
	}

	dialCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	dialer := net.Dialer{Timeout: cfg.ConnectTimeout}
	conn, err := dialer.DialContext(dialCtx, "tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)))
	if err != nil {
		mapped := mapTransportError(err, cfg.ConnectTimeout)
		d.recordFailure(mapped)
		return mapped
	}

	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		_ = conn.Close()
		return ErrClosed
	}
	d.conn = conn
	d.health.Status = protocol.HealthHealthy
	d.health.Message = ""
	d.health.LastCheck = time.Now()
	d.health.LastSuccess = time.Now()
	d.health.LastError = ""
	d.health.LastErrorAt = time.Time{}
	d.health.ConsecutiveFailures = 0
	d.mu.Unlock()
	return nil
}

// Read sends one request per point and parses the response fields declared in
// that point's binding.
func (d *Driver) Read(ctx context.Context, points []protocol.PointBinding) ([]protocol.PointSample, error) {
	if len(points) == 0 {
		return nil, nil
	}
	cfg := d.currentConfig()
	mappings := make([]pointMapping, len(points))
	for index, point := range points {
		mapping, err := parsePointBinding(point, cfg.Framing, cfg.MaxFrameSize)
		if err != nil {
			return nil, fmt.Errorf("read point %s: %w", point.PointIdentifier, err)
		}
		mappings[index] = mapping
	}

	d.mu.RLock()
	channelID := d.channelID
	d.mu.RUnlock()
	samples := make([]protocol.PointSample, 0, len(points))
	for index, point := range points {
		request, err := buildRequest(mappings[index], point, nil)
		if err != nil {
			return nil, fmt.Errorf("read point %s: %w", point.PointIdentifier, err)
		}
		payload, err := d.transact(ctx, mappings[index], request, true)
		if err != nil {
			return nil, fmt.Errorf("read point %s: %w", point.PointIdentifier, err)
		}
		values, err := decodeFields(payload, mappings[index].fields)
		if err != nil {
			return nil, fmt.Errorf("decode point %s: %w", point.PointIdentifier, err)
		}
		if len(values) == 0 {
			samples = append(samples, pointSample(point, channelID, "bytes", append([]byte(nil), payload...)))
			continue
		}
		for _, value := range values {
			source, exact := matchPoint(points, value.Identifier)
			identifier := value.Identifier
			if exact {
				identifier = firstIdentifier(source.PointIdentifier, value.Identifier)
			}
			samples = append(samples, protocol.PointSample{
				ChannelID:       channelID,
				ProductID:       source.ProductID,
				ProductKey:      source.ProductKey,
				DeviceID:        source.DeviceID,
				DeviceNo:        source.DeviceNo,
				PointID:         source.PointID,
				PointIdentifier: identifier,
				PointKind:       source.PointKind,
				DataType:        value.DataType,
				Value:           value.Value,
				Quality:         protocol.QualityGood,
				Timestamp:       time.Now(),
			})
		}
	}
	return samples, nil
}

// Write publishes one framed request. If the binding declares fields, a
// response prefix, or expectResponse=true, the response is read and validated.
func (d *Driver) Write(ctx context.Context, point protocol.PointBinding, value any) error {
	cfg := d.currentConfig()
	mapping, err := parsePointBinding(point, cfg.Framing, cfg.MaxFrameSize)
	if err != nil {
		return err
	}
	if point.ReadOnly {
		return fmt.Errorf("%w: point %s is read-only", ErrInvalidBinding, point.PointIdentifier)
	}
	request, err := buildRequest(mapping, point, value)
	if err != nil {
		return err
	}
	_, err = d.transact(ctx, mapping, request, mapping.expectResponse)
	return err
}

// Health returns a copy of the current connection health.
func (d *Driver) Health() protocol.DriverHealth {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.health
}

// Close closes the TCP connection. It is safe to call more than once.
func (d *Driver) Close() error {
	d.mu.Lock()
	conn := d.conn
	d.conn = nil
	d.closed = true
	d.health.Status = protocol.HealthStopped
	d.health.Message = "stopped"
	d.health.Since = time.Now()
	d.health.LastCheck = time.Now()
	d.mu.Unlock()
	if conn == nil {
		return nil
	}
	err := conn.Close()
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (d *Driver) transact(ctx context.Context, mapping pointMapping, payload []byte, expectResponse bool) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	d.opMu.Lock()
	defer d.opMu.Unlock()

	conn, cfg, err := d.connectionForOperation(ctx)
	if err != nil {
		return nil, err
	}

	requestPayload, err := appendChecksum(payload, mapping.checksum)
	if err != nil {
		return nil, err
	}
	requestFrame, err := mapping.framing.encodePayload(requestPayload, cfg.MaxFrameSize)
	if err != nil {
		return nil, err
	}

	_ = conn.SetWriteDeadline(requestDeadline(ctx, cfg.WriteTimeout))
	if err := writeFull(conn, requestFrame); err != nil {
		mapped := mapTransportError(err, cfg.WriteTimeout)
		d.dropConnection(conn, mapped)
		return nil, mapped
	}
	if !expectResponse {
		_ = conn.SetDeadline(time.Time{})
		d.recordSuccess()
		return nil, nil
	}

	_ = conn.SetReadDeadline(requestDeadline(ctx, cfg.ReadTimeout))
	responseFrame, err := mapping.framing.readFrame(conn, cfg.MaxFrameSize)
	if err != nil {
		mapped := mapTransportError(err, cfg.ReadTimeout)
		d.dropConnection(conn, mapped)
		return nil, mapped
	}
	_ = conn.SetDeadline(time.Time{})

	responsePayload, err := verifyChecksum(responseFrame, mapping.checksum)
	if err != nil {
		d.recordFailure(err)
		return nil, err
	}
	if len(mapping.matchPrefix) > 0 && !bytes.HasPrefix(responsePayload, mapping.matchPrefix) {
		err := fmt.Errorf("%w: expected prefix %x", ErrResponseMismatch, mapping.matchPrefix)
		d.recordFailure(err)
		return nil, err
	}
	d.recordSuccess()
	return responsePayload, nil
}

func (d *Driver) connectionForOperation(ctx context.Context) (net.Conn, Config, error) {
	d.mu.RLock()
	conn := d.conn
	closed := d.closed
	cfg := d.config
	channelID := d.channelID
	d.mu.RUnlock()
	if closed {
		return nil, Config{}, ErrClosed
	}
	if conn != nil {
		return conn, cfg, nil
	}

	var lastErr error
	for attempt := 0; attempt <= cfg.MaxReconnectAttempts; attempt++ {
		if attempt > 0 && !sleepContext(ctx, cfg.ReconnectDelay) {
			return nil, Config{}, ctx.Err()
		}
		if err := d.connect(ctx, channelID, cfg, false); err != nil {
			lastErr = err
			continue
		}
		d.mu.RLock()
		conn = d.conn
		closed = d.closed
		d.mu.RUnlock()
		if closed {
			return nil, Config{}, ErrClosed
		}
		if conn != nil {
			return conn, cfg, nil
		}
		lastErr = ErrNotConnected
	}
	if lastErr == nil {
		lastErr = ErrNotConnected
	}
	return nil, Config{}, lastErr
}

func (d *Driver) dropConnection(conn net.Conn, err error) {
	d.mu.Lock()
	if d.conn == conn {
		d.conn = nil
		d.recordFailureLocked(err)
		d.mu.Unlock()
		_ = conn.Close()
		return
	}
	d.mu.Unlock()
}

func (d *Driver) currentConfig() Config {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config
}

func (d *Driver) recordSuccess() {
	now := time.Now()
	d.mu.Lock()
	d.health.Status = protocol.HealthHealthy
	d.health.Message = ""
	d.health.LastCheck = now
	d.health.LastSuccess = now
	d.health.LastError = ""
	d.health.LastErrorAt = time.Time{}
	d.health.ConsecutiveFailures = 0
	d.mu.Unlock()
}

func (d *Driver) recordFailure(err error) {
	d.mu.Lock()
	d.recordFailureLocked(err)
	d.mu.Unlock()
}

func (d *Driver) recordFailureLocked(err error) {
	now := time.Now()
	d.health.Status = protocol.HealthDegraded
	d.health.Message = err.Error()
	d.health.LastCheck = now
	d.health.LastError = err.Error()
	d.health.LastErrorAt = now
	d.health.ConsecutiveFailures++
}

func pointSample(point protocol.PointBinding, channelID, dataType string, value any) protocol.PointSample {
	return protocol.PointSample{
		ChannelID:       channelID,
		ProductID:       point.ProductID,
		ProductKey:      point.ProductKey,
		DeviceID:        point.DeviceID,
		DeviceNo:        point.DeviceNo,
		PointID:         point.PointID,
		PointIdentifier: point.PointIdentifier,
		PointKind:       point.PointKind,
		DataType:        dataType,
		Value:           value,
		Quality:         protocol.QualityGood,
		Timestamp:       time.Now(),
	}
}

func matchPoint(points []protocol.PointBinding, identifier string) (protocol.PointBinding, bool) {
	for _, point := range points {
		if point.PointIdentifier == identifier {
			return point, true
		}
	}
	if len(points) > 0 {
		return points[0], false
	}
	return protocol.PointBinding{}, false
}

func firstIdentifier(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func requestDeadline(ctx context.Context, timeout time.Duration) time.Time {
	deadline := time.Now().Add(timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		return contextDeadline
	}
	return deadline
}

func sleepContext(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func mapTransportError(err error, timeout time.Duration) error {
	if errors.Is(err, ErrFrameTooLarge) || errors.Is(err, ErrInvalidFrame) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %v", ErrTimeout, err)
	}
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("%w: %w", ErrConnectionUpstream, err)
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("%w after %s: %v", ErrTimeout, timeout, err)
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed) {
		return fmt.Errorf("%w: %v", ErrConnectionClosed, err)
	}
	return fmt.Errorf("%w: %v", ErrConnectionUpstream, err)
}

func writeFull(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := writer.Write(data)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		data = data[written:]
	}
	return nil
}
