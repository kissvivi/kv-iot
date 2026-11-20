package modbus

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"kv-iot/datacenter/protocol"
)

const (
	mbapHeaderLength = 7
	maxPDULength     = 253
)

// Driver implements a Modbus TCP client for the generic ProtocolDriver
// contract. One driver instance owns one TCP connection.
type Driver struct {
	opMu sync.Mutex
	mu   sync.RWMutex

	conn          net.Conn
	config        Config
	channelID     string
	transactionID uint16
	closed        bool
	health        protocol.ChannelHealth
}

// New creates a Modbus TCP driver without opening a connection.
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

// Connect opens the configured Modbus TCP connection.
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

	d.opMu.Lock()
	defer d.opMu.Unlock()

	if d.conn != nil {
		_ = d.conn.Close()
		d.conn = nil
	}

	now := time.Now()
	d.mu.Lock()
	d.config = cfg
	d.channelID = config.ChannelID
	d.transactionID = 0
	d.closed = false
	d.health = protocol.ChannelHealth{
		ChannelID: config.ChannelID,
		Status:    protocol.HealthConnecting,
		Since:     now,
		LastCheck: now,
	}
	d.mu.Unlock()

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

// Read reads the requested points from the connected device.
func (d *Driver) Read(ctx context.Context, points []protocol.PointBinding) ([]protocol.PointSample, error) {
	if len(points) == 0 {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	d.opMu.Lock()
	defer d.opMu.Unlock()
	if err := d.ready(); err != nil {
		return nil, err
	}

	d.mu.RLock()
	cfg := d.config
	channelID := d.channelID
	d.mu.RUnlock()

	parsed := make([]pointMapping, len(points))
	for index, point := range points {
		mapping, err := parsePointBinding(point, cfg.UnitID)
		if err != nil {
			return nil, err
		}
		parsed[index] = mapping
	}

	now := time.Now()
	samples := make([]protocol.PointSample, 0, len(points))
	for index, point := range points {
		result, err := d.readPoint(ctx, parsed[index])
		if err != nil {
			return nil, fmt.Errorf("read point %s: %w", point.PointIdentifier, err)
		}
		value, err := decodeValue(parsed[index], result)
		if err != nil {
			return nil, fmt.Errorf("decode point %s: %w", point.PointIdentifier, err)
		}
		samples = append(samples, protocol.PointSample{
			ChannelID:       channelID,
			ProductID:       point.ProductID,
			ProductKey:      point.ProductKey,
			DeviceID:        point.DeviceID,
			DeviceNo:        point.DeviceNo,
			PointID:         point.PointID,
			PointIdentifier: point.PointIdentifier,
			PointKind:       point.PointKind,
			DataType:        parsed[index].dataType,
			Value:           value,
			Quality:         protocol.QualityGood,
			Timestamp:       now,
		})
	}
	d.recordSuccess()
	return samples, nil
}

// Write writes one coil or one holding register. Bit writes perform a
// read-modify-write against the holding register.
func (d *Driver) Write(ctx context.Context, point protocol.PointBinding, value any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if point.ReadOnly {
		return fmt.Errorf("%w: %s", ErrReadOnlyPoint, point.PointIdentifier)
	}

	d.opMu.Lock()
	defer d.opMu.Unlock()
	if err := d.ready(); err != nil {
		return err
	}

	d.mu.RLock()
	cfg := d.config
	d.mu.RUnlock()
	mapping, err := parsePointBinding(point, cfg.UnitID)
	if err != nil {
		return err
	}
	if !mapping.writable {
		return fmt.Errorf("%w: area %s", ErrUnsupportedArea, mapping.area)
	}

	switch mapping.area {
	case areaCoil:
		boolean, err := boolValue(value)
		if err != nil {
			return err
		}
		if err := d.writeSingleCoil(ctx, mapping, boolean); err != nil {
			return fmt.Errorf("write point %s: %w", point.PointIdentifier, err)
		}
	case areaHoldingRegister:
		if mapping.bitOffset != nil {
			boolean, err := bitWriteValue(value)
			if err != nil {
				return err
			}
			if err := d.writeBit(ctx, mapping, boolean); err != nil {
				return fmt.Errorf("write point %s: %w", point.PointIdentifier, err)
			}
			break
		}
		register, err := encodeSingleRegister(mapping, value)
		if err != nil {
			return err
		}
		if err := d.writeSingleRegister(ctx, mapping, register); err != nil {
			return fmt.Errorf("write point %s: %w", point.PointIdentifier, err)
		}
	default:
		return fmt.Errorf("%w: area %s", ErrUnsupportedArea, mapping.area)
	}

	d.recordSuccess()
	return nil
}

// Health returns a copy of the current channel health.
func (d *Driver) Health() protocol.DriverHealth {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.health
}

// Close closes the TCP connection. It is safe to call more than once.
func (d *Driver) Close() error {
	d.opMu.Lock()
	defer d.opMu.Unlock()

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
	return conn.Close()
}

func (d *Driver) ready() error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.closed || d.conn == nil {
		return ErrNotConnected
	}
	return nil
}

func (d *Driver) readPoint(ctx context.Context, mapping pointMapping) (readResult, error) {
	address, err := adjustedAddress(mapping.address, d.addressBase())
	if err != nil {
		return readResult{}, err
	}
	if mapping.area == areaCoil || mapping.area == areaDiscreteInput {
		bits, err := d.readBits(ctx, mapping.unitID, mapping.readFunc, address, uint16(mapping.quantity))
		return readResult{bits: bits}, err
	}
	registers, err := d.readRegisters(ctx, mapping.unitID, mapping.readFunc, address, uint16(mapping.quantity))
	return readResult{registers: registers}, err
}

func (d *Driver) addressBase() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config.AddressBase
}

func (d *Driver) readBits(ctx context.Context, unitID int, function byte, address, quantity uint16) ([]bool, error) {
	if quantity == 0 || quantity > 2000 {
		return nil, fmt.Errorf("%w: bit quantity must be between 1 and 2000", ErrInvalidBinding)
	}
	if uint32(address)+uint32(quantity)-1 > 65535 {
		return nil, fmt.Errorf("%w: bit range exceeds address space", ErrInvalidBinding)
	}
	pdu := []byte{
		function,
		byte(address >> 8), byte(address),
		byte(quantity >> 8), byte(quantity),
	}
	response, err := d.roundTrip(ctx, byte(unitID), pdu)
	if err != nil {
		return nil, err
	}
	byteCount := (int(quantity) + 7) / 8
	if len(response) != 2+byteCount || int(response[1]) != byteCount {
		return nil, d.invalidResponse("bit response length mismatch")
	}
	values := make([]bool, int(quantity))
	for index := range values {
		values[index] = response[2+index/8]&(1<<uint(index%8)) != 0
	}
	return values, nil
}

func (d *Driver) readRegisters(ctx context.Context, unitID int, function byte, address, quantity uint16) ([]uint16, error) {
	if quantity == 0 || quantity > 125 {
		return nil, fmt.Errorf("%w: register quantity must be between 1 and 125", ErrInvalidBinding)
	}
	if uint32(address)+uint32(quantity)-1 > 65535 {
		return nil, fmt.Errorf("%w: register range exceeds address space", ErrInvalidBinding)
	}
	pdu := []byte{
		function,
		byte(address >> 8), byte(address),
		byte(quantity >> 8), byte(quantity),
	}
	response, err := d.roundTrip(ctx, byte(unitID), pdu)
	if err != nil {
		return nil, err
	}
	byteCount := int(quantity) * 2
	if len(response) != 2+byteCount || int(response[1]) != byteCount {
		return nil, d.invalidResponse("register response length mismatch")
	}
	values := make([]uint16, int(quantity))
	for index := range values {
		values[index] = binary.BigEndian.Uint16(response[2+index*2:])
	}
	return values, nil
}

func (d *Driver) writeSingleCoil(ctx context.Context, mapping pointMapping, value bool) error {
	address, err := adjustedAddress(mapping.address, d.addressBase())
	if err != nil {
		return err
	}
	wireValue := uint16(0)
	if value {
		wireValue = 0xff00
	}
	pdu := []byte{
		fcWriteCoil,
		byte(address >> 8), byte(address),
		byte(wireValue >> 8), byte(wireValue),
	}
	response, err := d.roundTrip(ctx, byte(mapping.unitID), pdu)
	if err != nil {
		return err
	}
	if !bytes.Equal(response, pdu) {
		return d.invalidResponse("write coil response mismatch")
	}
	return nil
}

func (d *Driver) writeSingleRegister(ctx context.Context, mapping pointMapping, value uint16) error {
	address, err := adjustedAddress(mapping.address, d.addressBase())
	if err != nil {
		return err
	}
	pdu := []byte{
		fcWriteRegister,
		byte(address >> 8), byte(address),
		byte(value >> 8), byte(value),
	}
	response, err := d.roundTrip(ctx, byte(mapping.unitID), pdu)
	if err != nil {
		return err
	}
	if !bytes.Equal(response, pdu) {
		return d.invalidResponse("write register response mismatch")
	}
	return nil
}

func (d *Driver) writeBit(ctx context.Context, mapping pointMapping, value bool) error {
	address, err := adjustedAddress(mapping.address, d.addressBase())
	if err != nil {
		return err
	}
	registers, err := d.readRegisters(ctx, mapping.unitID, fcReadHolding, address, 1)
	if err != nil {
		return err
	}
	current := registers[0]
	mask := uint16(1) << uint(*mapping.bitOffset)
	if value {
		current |= mask
	} else {
		current &^= mask
	}
	return d.writeSingleRegister(ctx, mapping, current)
}

func (d *Driver) roundTrip(ctx context.Context, unitID byte, requestPDU []byte) ([]byte, error) {
	if len(requestPDU) == 0 || len(requestPDU) > maxPDULength {
		return nil, fmt.Errorf("%w: invalid request PDU length %d", ErrInvalidResponse, len(requestPDU))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	d.mu.Lock()
	if d.conn == nil {
		d.mu.Unlock()
		return nil, ErrNotConnected
	}
	conn := d.conn
	d.transactionID++
	transactionID := d.transactionID
	cfg := d.config
	d.mu.Unlock()

	request := make([]byte, mbapHeaderLength+len(requestPDU))
	binary.BigEndian.PutUint16(request[0:2], transactionID)
	binary.BigEndian.PutUint16(request[2:4], 0)
	binary.BigEndian.PutUint16(request[4:6], uint16(len(requestPDU)+1))
	request[6] = unitID
	copy(request[mbapHeaderLength:], requestPDU)

	_ = conn.SetWriteDeadline(requestDeadline(ctx, cfg.WriteTimeout))
	if err := writeFull(conn, request); err != nil {
		mapped := mapTransportError(err, cfg.WriteTimeout)
		d.dropConnection()
		d.recordFailure(mapped)
		return nil, mapped
	}

	_ = conn.SetReadDeadline(requestDeadline(ctx, cfg.ReadTimeout))
	var header [mbapHeaderLength]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		mapped := mapTransportError(err, cfg.ReadTimeout)
		d.dropConnection()
		d.recordFailure(mapped)
		return nil, mapped
	}

	responseTransactionID := binary.BigEndian.Uint16(header[0:2])
	protocolID := binary.BigEndian.Uint16(header[2:4])
	length := int(binary.BigEndian.Uint16(header[4:6]))
	responseUnitID := header[6]
	if protocolID != 0 || length < 2 || length > 254 ||
		responseTransactionID != transactionID || responseUnitID != unitID {
		return nil, d.invalidResponse("invalid MBAP header")
	}

	responsePDU := make([]byte, length-1)
	if _, err := io.ReadFull(conn, responsePDU); err != nil {
		mapped := mapTransportError(err, cfg.ReadTimeout)
		d.dropConnection()
		d.recordFailure(mapped)
		return nil, mapped
	}
	_ = conn.SetDeadline(time.Time{})

	if len(responsePDU) == 2 && responsePDU[0] == requestPDU[0]|0x80 {
		exception := &Exception{FunctionCode: requestPDU[0], Code: responsePDU[1]}
		d.recordFailure(exception)
		return nil, exception
	}
	if len(responsePDU) == 0 || responsePDU[0] != requestPDU[0] {
		return nil, d.invalidResponse("function code mismatch")
	}
	return responsePDU, nil
}

func (d *Driver) invalidResponse(message string) error {
	err := fmt.Errorf("%w: %s", ErrInvalidResponse, message)
	d.dropConnection()
	d.recordFailure(err)
	return err
}

func (d *Driver) dropConnection() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.conn != nil {
		_ = d.conn.Close()
		d.conn = nil
	}
}

func (d *Driver) recordSuccess() {
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	d.health.Status = protocol.HealthHealthy
	d.health.Message = ""
	d.health.LastCheck = now
	d.health.LastSuccess = now
	d.health.LastError = ""
	d.health.LastErrorAt = time.Time{}
	d.health.ConsecutiveFailures = 0
}

func (d *Driver) recordFailure(err error) {
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	d.health.Status = protocol.HealthDegraded
	d.health.Message = err.Error()
	d.health.LastCheck = now
	d.health.LastError = err.Error()
	d.health.LastErrorAt = now
	d.health.ConsecutiveFailures++
}

func adjustedAddress(address, addressBase int) (uint16, error) {
	adjusted := address - addressBase
	if adjusted < 0 || adjusted > 65535 {
		return 0, fmt.Errorf("%w: address %d with addressBase %d is out of range", ErrInvalidBinding, address, addressBase)
	}
	return uint16(adjusted), nil
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
