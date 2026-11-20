package tcpraw

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

type pendingResponse struct {
	chunks chan []byte
	errCh  chan error
}

func newPendingResponse() *pendingResponse {
	return &pendingResponse{
		chunks: make(chan []byte, 64),
		errCh:  make(chan error, 1),
	}
}

func (p *pendingResponse) fail(err error) {
	select {
	case p.errCh <- err:
	default:
	}
}

// Driver implements one persistent TCP raw connection. A background reader
// routes bytes to the active request or to stream subscribers. Raw protocols
// have no inherent message boundary, so a request and an unsolicited stream
// cannot be distinguished if they interleave.
type Driver struct {
	connectMu sync.Mutex
	writeMu   sync.Mutex
	mu        sync.RWMutex

	conn           net.Conn
	down           chan struct{}
	config         Config
	channelID      string
	closed         bool
	health         protocol.ChannelHealth
	pending        *pendingResponse
	subscribers    map[uint64]func([]byte)
	nextSubscriber uint64
}

var _ protocol.RawStreamDriver = (*Driver)(nil)

// New creates a TCP raw driver without opening a connection.
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
		subscribers: make(map[uint64]func([]byte)),
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
	oldDown := d.down
	d.conn = nil
	d.down = nil
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
	closeDown(oldDown)

	dialCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	dialer := net.Dialer{Timeout: cfg.ConnectTimeout}
	conn, err := dialer.DialContext(dialCtx, "tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)))
	if err != nil {
		mapped := mapTransportError(err, cfg.ConnectTimeout)
		d.recordFailure(mapped, false)
		return mapped
	}

	down := make(chan struct{})
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		_ = conn.Close()
		return ErrClosed
	}
	d.conn = conn
	d.down = down
	d.health.Status = protocol.HealthHealthy
	d.health.Message = ""
	d.health.LastCheck = time.Now()
	d.health.LastSuccess = time.Now()
	d.health.LastError = ""
	d.health.LastErrorAt = time.Time{}
	d.health.ConsecutiveFailures = 0
	d.mu.Unlock()

	go d.readLoop(conn, down, readBufferSize(cfg))
	return nil
}

// SendRaw writes one payload and waits for one response according to the
// configured response mode.
func (d *Driver) SendRaw(ctx context.Context, payload []byte) ([]byte, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("%w: payload is required", ErrInvalidBinding)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	conn, cfg, err := d.connectionForOperation(ctx)
	if err != nil {
		return nil, err
	}

	pending := newPendingResponse()
	d.mu.Lock()
	if d.conn != conn {
		d.mu.Unlock()
		return nil, ErrNotConnected
	}
	d.pending = pending
	d.mu.Unlock()

	_ = conn.SetWriteDeadline(requestDeadline(ctx, cfg.WriteTimeout))
	if err := writeFull(conn, payload); err != nil {
		d.clearPending(pending)
		mapped := mapTransportError(err, cfg.WriteTimeout)
		d.dropConnection(conn, mapped)
		return nil, mapped
	}

	response, err := d.awaitResponse(ctx, pending, cfg)
	d.clearPending(pending)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			d.dropConnection(conn, err)
		}
		return nil, err
	}
	d.recordSuccess()
	return response, nil
}

// Subscribe registers a byte-stream callback and blocks until ctx is canceled.
// The driver reconnects automatically after a connection loss.
func (d *Driver) Subscribe(ctx context.Context, handler func([]byte)) error {
	if handler == nil {
		return fmt.Errorf("%w: stream handler is required", ErrInvalidBinding)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	subscriberID := d.addSubscriber(handler)
	defer d.removeSubscriber(subscriberID)

	for {
		_, _, err := d.connectionForOperation(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if !sleepContext(ctx, d.currentConfig().ReconnectDelay) {
				return ctx.Err()
			}
			continue
		}

		down := d.connectionDown()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-down:
			if !sleepContext(ctx, d.currentConfig().ReconnectDelay) {
				return ctx.Err()
			}
		}
	}
}

// Read sends each point's raw request and returns the response bytes.
func (d *Driver) Read(ctx context.Context, points []protocol.PointBinding) ([]protocol.PointSample, error) {
	if len(points) == 0 {
		return nil, nil
	}
	now := time.Now()
	d.mu.RLock()
	channelID := d.channelID
	d.mu.RUnlock()

	samples := make([]protocol.PointSample, 0, len(points))
	for _, point := range points {
		if err := point.Validate(); err != nil {
			return nil, err
		}
		request, err := requestPayload(point.Config)
		if err != nil {
			return nil, fmt.Errorf("point %s: %w", point.PointIdentifier, err)
		}
		response, err := d.SendRaw(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("read point %s: %w", point.PointIdentifier, err)
		}
		samples = append(samples, bindingSample(point, channelID, now, response))
	}
	return samples, nil
}

// Write sends a raw payload. The value may be []byte, a string, or omitted
// when the point binding provides request/requestHex.
func (d *Driver) Write(ctx context.Context, point protocol.PointBinding, value any) error {
	if err := point.Validate(); err != nil {
		return err
	}
	if point.ReadOnly {
		return fmt.Errorf("%w: point %s is read-only", ErrInvalidBinding, point.PointIdentifier)
	}
	payload, err := bytesValue(value)
	if err != nil {
		return err
	}
	if len(payload) == 0 {
		payload, err = requestPayload(point.Config)
		if err != nil {
			return err
		}
	}
	_, err = d.SendRaw(ctx, payload)
	return err
}

// Health returns a copy of the current connection health.
func (d *Driver) Health() protocol.DriverHealth {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.health
}

// Close stops the connection and unblocks subscribers.
func (d *Driver) Close() error {
	d.mu.Lock()
	conn := d.conn
	down := d.down
	d.conn = nil
	d.down = nil
	d.closed = true
	d.health.Status = protocol.HealthStopped
	d.health.Message = "stopped"
	d.health.Since = time.Now()
	d.health.LastCheck = time.Now()
	d.mu.Unlock()

	closeDown(down)
	if conn == nil {
		return nil
	}
	err := conn.Close()
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (d *Driver) connectionForOperation(ctx context.Context) (net.Conn, Config, error) {
	d.mu.RLock()
	conn := d.conn
	closed := d.closed
	cfg := d.config
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
		if err := d.connect(ctx, d.channelIDValue(), cfg, false); err != nil {
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

func (d *Driver) awaitResponse(ctx context.Context, pending *pendingResponse, cfg Config) ([]byte, error) {
	timer := time.NewTimer(cfg.ResponseTimeout)
	defer timer.Stop()

	buffer := make([]byte, 0, cfg.MaxReadBytes)
	for {
		select {
		case chunk := <-pending.chunks:
			if len(buffer)+len(chunk) > cfg.MaxResponseBytes {
				return nil, fmt.Errorf("%w: limit %d", ErrResponseTooLarge, cfg.MaxResponseBytes)
			}
			buffer = append(buffer, chunk...)
			if complete, response := responseComplete(buffer, cfg); complete {
				return response, nil
			}
		case err := <-pending.errCh:
			return nil, err
		case <-timer.C:
			return nil, fmt.Errorf("%w after %s", ErrTimeout, cfg.ResponseTimeout)
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("%w: %v", ErrTimeout, ctx.Err())
			}
			return nil, ctx.Err()
		}
	}
}

func responseComplete(buffer []byte, cfg Config) (bool, []byte) {
	switch cfg.ResponseMode {
	case ResponseModeChunk:
		return true, append([]byte(nil), buffer...)
	case ResponseModeFixed:
		if len(buffer) < cfg.ResponseLength {
			return false, nil
		}
		return true, append([]byte(nil), buffer[:cfg.ResponseLength]...)
	case ResponseModeDelimiter:
		index := bytes.Index(buffer, cfg.ResponseDelimiter)
		if index < 0 {
			return false, nil
		}
		end := index + len(cfg.ResponseDelimiter)
		return true, append([]byte(nil), buffer[:end]...)
	default:
		return true, append([]byte(nil), buffer...)
	}
}

func readBufferSize(cfg Config) int {
	if cfg.ResponseMode == ResponseModeFixed && cfg.ResponseLength > 0 && cfg.ResponseLength < cfg.MaxReadBytes {
		return cfg.ResponseLength
	}
	return cfg.MaxReadBytes
}

func (d *Driver) readLoop(conn net.Conn, down chan struct{}, maxReadBytes int) {
	buffer := make([]byte, maxReadBytes)
	for {
		count, err := conn.Read(buffer)
		if count > 0 {
			d.dispatch(append([]byte(nil), buffer[:count]...))
		}
		if err != nil {
			mapped := mapTransportError(err, d.currentConfig().ReadTimeout)
			d.failRead(conn, down, mapped)
			return
		}
	}
}

func (d *Driver) dispatch(payload []byte) {
	d.mu.RLock()
	pending := d.pending
	handlers := make([]func([]byte), 0, len(d.subscribers))
	for _, handler := range d.subscribers {
		handlers = append(handlers, handler)
	}
	d.mu.RUnlock()

	if pending != nil {
		select {
		case pending.chunks <- payload:
			return
		default:
			pending.fail(fmt.Errorf("%w: response queue overflow", ErrInvalidResponse))
			return
		}
	}
	for _, handler := range handlers {
		handler(append([]byte(nil), payload...))
	}
}

func (d *Driver) failRead(conn net.Conn, down chan struct{}, err error) {
	d.mu.Lock()
	if d.conn != conn {
		d.mu.Unlock()
		return
	}
	d.conn = nil
	d.down = nil
	pending := d.pending
	if pending != nil {
		pending.fail(err)
	}
	closed := d.closed
	if !closed {
		d.recordFailureLocked(err)
	}
	d.mu.Unlock()

	closeDown(down)
	_ = conn.Close()
}

func (d *Driver) dropConnection(conn net.Conn, err error) {
	d.mu.Lock()
	if d.conn == conn {
		d.conn = nil
		down := d.down
		d.down = nil
		d.recordFailureLocked(err)
		d.mu.Unlock()
		closeDown(down)
		_ = conn.Close()
		return
	}
	d.mu.Unlock()
}

func (d *Driver) clearPending(pending *pendingResponse) {
	d.mu.Lock()
	if d.pending == pending {
		d.pending = nil
	}
	d.mu.Unlock()
}

func (d *Driver) addSubscriber(handler func([]byte)) uint64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.nextSubscriber++
	id := d.nextSubscriber
	d.subscribers[id] = handler
	return id
}

func (d *Driver) removeSubscriber(id uint64) {
	d.mu.Lock()
	delete(d.subscribers, id)
	d.mu.Unlock()
}

func (d *Driver) connectionDown() <-chan struct{} {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.down == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return d.down
}

func (d *Driver) channelIDValue() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.channelID
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

func (d *Driver) recordFailure(err error, connecting bool) {
	d.mu.Lock()
	d.recordFailureLocked(err)
	if connecting {
		d.health.Status = protocol.HealthConnecting
	}
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

func closeDown(ch chan struct{}) {
	if ch == nil {
		return
	}
	select {
	case <-ch:
	default:
		close(ch)
	}
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

func requestDeadline(ctx context.Context, timeout time.Duration) time.Time {
	deadline := time.Now().Add(timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		return contextDeadline
	}
	return deadline
}

func mapTransportError(err error, timeout time.Duration) error {
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
