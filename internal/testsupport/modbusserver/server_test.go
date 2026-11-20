package modbusserver

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

func TestReadHoldingRegisters(t *testing.T) {
	server := New(Config{HoldingRegisters: 16})
	if err := server.Data().SetHoldingRegister(2, 0x1234); err != nil {
		t.Fatalf("seed register: %v", err)
	}
	if err := server.Data().SetHoldingRegister(3, 0x5678); err != nil {
		t.Fatalf("seed register: %v", err)
	}
	startServer(t, server)

	conn := dialServer(t, server.Addr())
	defer conn.Close()

	response := roundTrip(t, conn, []byte{functionReadHoldingRegisters, 0, 2, 0, 2})
	if len(response) != 6 {
		t.Fatalf("response length = %d, want 6: %x", len(response), response)
	}
	if response[0] != functionReadHoldingRegisters || response[1] != 4 {
		t.Fatalf("response header = %x", response[:2])
	}
	if got := binary.BigEndian.Uint16(response[2:4]); got != 0x1234 {
		t.Fatalf("register 2 = %#x", got)
	}
	if got := binary.BigEndian.Uint16(response[4:6]); got != 0x5678 {
		t.Fatalf("register 3 = %#x", got)
	}
}

func TestWriteSingleRegister(t *testing.T) {
	server := New(Config{HoldingRegisters: 8})
	startServer(t, server)

	conn := dialServer(t, server.Addr())
	defer conn.Close()

	request := []byte{functionWriteSingleRegister, 0, 4, 0xAB, 0xCD}
	response := roundTrip(t, conn, request)
	if string(response) != string(request) {
		t.Fatalf("write response = %x, want %x", response, request)
	}
	value, ok := server.Data().HoldingRegister(4)
	if !ok || value != 0xABCD {
		t.Fatalf("register 4 = %#x, %v", value, ok)
	}
}

func TestWriteMultipleCoils(t *testing.T) {
	server := New(Config{Coils: 16})
	startServer(t, server)

	conn := dialServer(t, server.Addr())
	defer conn.Close()

	request := []byte{functionWriteMultipleCoils, 0, 3, 0, 10, 2, 0b01010101, 0b00000011}
	response := roundTrip(t, conn, request)
	if len(response) != 5 || response[0] != functionWriteMultipleCoils {
		t.Fatalf("write coils response = %x", response)
	}
	for i := uint16(0); i < 10; i++ {
		got, ok := server.Data().Coil(3 + i)
		if !ok {
			t.Fatalf("coil %d missing", 3+i)
		}
		want := i < 8 && i%2 == 0 || i >= 8
		if got != want {
			t.Fatalf("coil %d = %v, want %v", 3+i, got, want)
		}
	}
}

func TestIllegalAddressReturnsException(t *testing.T) {
	server := New(Config{HoldingRegisters: 4})
	startServer(t, server)

	conn := dialServer(t, server.Addr())
	defer conn.Close()

	response := roundTrip(t, conn, []byte{functionReadHoldingRegisters, 0, 3, 0, 2})
	want := []byte{functionReadHoldingRegisters | 0x80, exceptionIllegalAddress}
	if string(response) != string(want) {
		t.Fatalf("exception = %x, want %x", response, want)
	}
}

func TestDataRangesAreIndependent(t *testing.T) {
	server := New(Config{
		Coils:            2,
		DiscreteInputs:   3,
		HoldingRegisters: 4,
		InputRegisters:   5,
	})
	if err := server.Data().SetCoil(20, true); err == nil {
		t.Fatal("expected coil range error")
	}
	if err := server.Data().SetDiscreteInput(2, true); err != nil {
		t.Fatalf("set discrete input: %v", err)
	}
	if err := server.Data().SetInputRegister(4, 42); err != nil {
		t.Fatalf("set input register: %v", err)
	}
	if err := server.Data().SetHoldingRegister(4, 1); err == nil {
		t.Fatal("expected holding register range error")
	}
}

func startServer(t *testing.T, server *Server) {
	t.Helper()
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("close server: %v", err)
		}
	})
}

func dialServer(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	return conn
}

func roundTrip(t *testing.T, conn net.Conn, pdu []byte) []byte {
	t.Helper()
	request := make([]byte, 7+len(pdu))
	binary.BigEndian.PutUint16(request[0:2], 1)
	binary.BigEndian.PutUint16(request[4:6], uint16(len(pdu)+1))
	request[6] = 1
	copy(request[7:], pdu)
	if _, err := conn.Write(request); err != nil {
		t.Fatalf("write request: %v", err)
	}

	var header [7]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		t.Fatalf("read response header: %v", err)
	}
	length := binary.BigEndian.Uint16(header[4:6])
	if length < 2 {
		t.Fatalf("invalid response length %d", length)
	}
	response := make([]byte, int(length)-1)
	if _, err := io.ReadFull(conn, response); err != nil {
		t.Fatalf("read response PDU: %v", err)
	}
	return response
}
