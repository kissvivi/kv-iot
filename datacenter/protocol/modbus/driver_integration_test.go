package modbus

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"reflect"
	"strconv"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
	"kv-iot/internal/testsupport/modbusserver"
)

func TestDriverReadsCommonAreasAndTypes(t *testing.T) {
	server := modbusserver.New(modbusserver.Config{
		Coils:            16,
		DiscreteInputs:   16,
		HoldingRegisters: 16,
		InputRegisters:   16,
	})
	seedRegister(t, server, 0, 0x1234)
	seedRegister(t, server, 1, 0x5678)
	seedRegister(t, server, 2, 0x0008)
	seedRegister(t, server, 3, 1000)
	seedRegister(t, server, 4, 0xfffe)
	if err := server.Data().SetInputRegister(2, 4242); err != nil {
		t.Fatalf("seed input register: %v", err)
	}
	if err := server.Data().SetCoil(4, true); err != nil {
		t.Fatalf("seed coil: %v", err)
	}
	if err := server.Data().SetDiscreteInput(3, true); err != nil {
		t.Fatalf("seed discrete input: %v", err)
	}
	driver := startConnectedDriver(t, server, 0)

	points := []protocol.PointBinding{
		testPoint("coil", `{"area":"coil","address":4}`),
		testPoint("discrete", `{"area":"discrete_input","address":3}`),
		testPoint("holding", `{"area":"holding_register","address":0,"dataType":"uint16"}`),
		testPoint("input", `{"area":"input_register","address":2,"dataType":"uint16"}`),
		testPoint("uint32", `{"area":"holding_register","address":0,"dataType":"uint32"}`),
		testPoint("bit", `{"area":"holding_register","address":2,"dataType":"bool","bitOffset":3}`),
		testPoint("scaled", `{"area":"holding_register","address":3,"dataType":"uint16","scale":0.1}`),
		testPoint("int16", `{"area":"holding_register","address":4,"dataType":"int16"}`),
	}

	samples, err := driver.Read(context.Background(), points)
	if err != nil {
		t.Fatalf("read points: %v", err)
	}
	if len(samples) != len(points) {
		t.Fatalf("sample count = %d, want %d", len(samples), len(points))
	}
	wants := []any{
		true,
		true,
		uint16(0x1234),
		uint16(4242),
		uint32(0x12345678),
		true,
		float64(100),
		int16(-2),
	}
	for index, want := range wants {
		if !reflect.DeepEqual(samples[index].Value, want) {
			t.Errorf("sample %s = %#v (%T), want %#v (%T)",
				samples[index].PointIdentifier, samples[index].Value, samples[index].Value, want, want)
		}
		if samples[index].Quality != protocol.QualityGood {
			t.Errorf("sample %s quality = %s", samples[index].PointIdentifier, samples[index].Quality)
		}
	}
}

func TestDriverAddressBase(t *testing.T) {
	server := modbusserver.New(modbusserver.Config{HoldingRegisters: 8})
	seedRegister(t, server, 1, 77)
	driver := startConnectedDriver(t, server, 1)

	samples, err := driver.Read(context.Background(), []protocol.PointBinding{
		testPoint("based", `{"area":"holding_register","address":2,"dataType":"uint16"}`),
	})
	if err != nil {
		t.Fatalf("read with address base: %v", err)
	}
	if got := samples[0].Value; got != uint16(77) {
		t.Fatalf("value = %#v, want 77", got)
	}
}

func TestDriverWritesCoilsRegistersAndBits(t *testing.T) {
	server := modbusserver.New(modbusserver.Config{
		Coils:            16,
		HoldingRegisters: 16,
	})
	seedRegister(t, server, 6, 0b1010)
	driver := startConnectedDriver(t, server, 0)

	ctx := context.Background()
	if err := driver.Write(ctx, testPoint("coil", `{"area":"coil","address":2}`), true); err != nil {
		t.Fatalf("write coil: %v", err)
	}
	if value, ok := server.Data().Coil(2); !ok || !value {
		t.Fatalf("coil 2 = %v, %v", value, ok)
	}

	if err := driver.Write(ctx, testPoint("register", `{"area":"holding_register","address":3,"dataType":"uint16"}`), 4660); err != nil {
		t.Fatalf("write register: %v", err)
	}
	assertRegister(t, server, 3, 4660)

	if err := driver.Write(ctx, testPoint("scaled", `{"area":"holding_register","address":4,"dataType":"uint16","scale":0.1}`), 12.3); err != nil {
		t.Fatalf("write scaled register: %v", err)
	}
	assertRegister(t, server, 4, 123)

	if err := driver.Write(ctx, testPoint("little", `{"area":"holding_register","address":5,"dataType":"uint16","byteOrder":"little"}`), 0x1234); err != nil {
		t.Fatalf("write little-endian register: %v", err)
	}
	assertRegister(t, server, 5, 0x3412)

	bitPoint := testPoint("bit", `{"area":"holding_register","address":6,"dataType":"bool","bitOffset":1}`)
	if err := driver.Write(ctx, bitPoint, true); err != nil {
		t.Fatalf("set bit: %v", err)
	}
	assertRegister(t, server, 6, 0b1010)
	if err := driver.Write(ctx, bitPoint, false); err != nil {
		t.Fatalf("clear bit: %v", err)
	}
	assertRegister(t, server, 6, 0b1000)
}

func TestDriverMapsExceptionsAndKeepsConnection(t *testing.T) {
	server := modbusserver.New(modbusserver.Config{HoldingRegisters: 4})
	seedRegister(t, server, 0, 9)
	driver := startConnectedDriver(t, server, 0)

	_, err := driver.Read(context.Background(), []protocol.PointBinding{
		testPoint("missing", `{"area":"holding_register","address":4,"dataType":"uint16"}`),
	})
	if !errors.Is(err, ErrIllegalAddress) {
		t.Fatalf("read error = %v, want ErrIllegalAddress", err)
	}

	samples, err := driver.Read(context.Background(), []protocol.PointBinding{
		testPoint("valid", `{"area":"holding_register","address":0,"dataType":"uint16"}`),
	})
	if err != nil {
		t.Fatalf("read after exception: %v", err)
	}
	if got := samples[0].Value; got != uint16(9) {
		t.Fatalf("value after exception = %#v", got)
	}
}

func TestDriverRejectsReadOnlyAndUnsupportedWrites(t *testing.T) {
	server := modbusserver.New(modbusserver.Config{})
	driver := startConnectedDriver(t, server, 0)

	readOnly := testPoint("readonly", `{"area":"holding_register","address":0,"dataType":"uint16"}`)
	readOnly.ReadOnly = true
	if err := driver.Write(context.Background(), readOnly, 1); !errors.Is(err, ErrReadOnlyPoint) {
		t.Fatalf("read-only write error = %v", err)
	}

	if err := driver.Write(context.Background(),
		testPoint("input", `{"area":"input_register","address":0,"dataType":"uint16"}`), 1); !errors.Is(err, ErrUnsupportedArea) {
		t.Fatalf("input-register write error = %v", err)
	}
}

func TestDriverCloseAndNotConnected(t *testing.T) {
	server := modbusserver.New(modbusserver.Config{})
	driver := startConnectedDriver(t, server, 0)
	if err := driver.Close(); err != nil {
		t.Fatalf("close driver: %v", err)
	}
	if err := driver.Close(); err != nil {
		t.Fatalf("close driver again: %v", err)
	}
	_, err := driver.Read(context.Background(), []protocol.PointBinding{
		testPoint("point", `{"area":"coil","address":0}`),
	})
	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("read after close error = %v, want ErrNotConnected", err)
	}
	if driver.Health().Status != protocol.HealthStopped {
		t.Fatalf("health status = %s", driver.Health().Status)
	}
}

func TestTransportErrorMapping(t *testing.T) {
	if err := mapTransportError(timeoutTestError{}, time.Second); !errors.Is(err, ErrTimeout) {
		t.Fatalf("timeout mapping = %v", err)
	}
	if err := mapTransportError(net.ErrClosed, time.Second); !errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("closed mapping = %v", err)
	}
}

type timeoutTestError struct{}

func (timeoutTestError) Error() string   { return "test timeout" }
func (timeoutTestError) Timeout() bool   { return true }
func (timeoutTestError) Temporary() bool { return true }

func startConnectedDriver(t *testing.T, server *modbusserver.Server, addressBase int) *Driver {
	t.Helper()
	if err := server.Start(); err != nil {
		t.Fatalf("start modbus server: %v", err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("close modbus server: %v", err)
		}
	})

	host, portText, err := net.SplitHostPort(server.Addr())
	if err != nil {
		t.Fatalf("split server address: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parse server port: %v", err)
	}
	connectionConfig, err := json.Marshal(map[string]any{
		"host":           host,
		"port":           port,
		"unitId":         1,
		"addressBase":    addressBase,
		"connectTimeout": "1s",
		"readTimeout":    "1s",
		"writeTimeout":   "1s",
	})
	if err != nil {
		t.Fatalf("marshal connection config: %v", err)
	}
	driver, err := New(protocol.DriverConfig{
		ChannelID:    "test-channel",
		ProtocolType: protocol.ProtocolModbusTCP,
		Config:       connectionConfig,
	})
	if err != nil {
		t.Fatalf("create driver: %v", err)
	}
	t.Cleanup(func() {
		_ = driver.Close()
	})
	if err := driver.Connect(context.Background(), protocol.DriverConfig{
		ChannelID:    "test-channel",
		ProtocolType: protocol.ProtocolModbusTCP,
		Config:       connectionConfig,
	}); err != nil {
		t.Fatalf("connect driver: %v", err)
	}
	return driver
}

func testPoint(identifier, config string) protocol.PointBinding {
	return protocol.PointBinding{
		PointID:         int64(len(identifier)),
		PointIdentifier: identifier,
		Config:          json.RawMessage(config),
	}
}

func seedRegister(t *testing.T, server *modbusserver.Server, address uint16, value uint16) {
	t.Helper()
	if err := server.Data().SetHoldingRegister(address, value); err != nil {
		t.Fatalf("seed holding register %d: %v", address, err)
	}
}

func assertRegister(t *testing.T, server *modbusserver.Server, address uint16, want uint16) {
	t.Helper()
	got, ok := server.Data().HoldingRegister(address)
	if !ok {
		t.Fatalf("holding register %d missing", address)
	}
	if got != want {
		t.Fatalf("holding register %d = 0x%04x, want 0x%04x", address, got, want)
	}
}
