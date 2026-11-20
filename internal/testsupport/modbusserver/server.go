// Package modbusserver provides an in-process Modbus TCP test server.
package modbusserver

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"kv-iot/internal/testsupport/tcpserver"
)

// Config controls a Modbus TCP test server.
type Config struct {
	Addr string

	Coils            int
	DiscreteInputs   int
	HoldingRegisters int
	InputRegisters   int
}

// Server is a startable Modbus TCP server with thread-safe in-memory data.
type Server struct {
	data *Data
	base *tcpserver.Server
}

// New creates a Modbus TCP test server. Address-space sizes default to 65536.
func New(cfg Config) *Server {
	server := &Server{data: newData(cfg)}
	server.base = tcpserver.NewWithConfig(tcpserver.Config{Addr: cfg.Addr}, server.handleConnection)
	return server
}

// Start begins listening.
func (s *Server) Start() error {
	return s.base.Start()
}

// Addr returns the actual listening address after Start.
func (s *Server) Addr() string {
	return s.base.Addr()
}

// Close stops the server.
func (s *Server) Close() error {
	return s.base.Close()
}

// Data exposes the in-memory address spaces for test setup and assertions.
func (s *Server) Data() *Data {
	return s.data
}

func (s *Server) handleConnection(_ context.Context, conn net.Conn) error {
	var header [7]byte
	for {
		if _, err := io.ReadFull(conn, header[:]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}

		transactionID := binary.BigEndian.Uint16(header[0:2])
		protocolID := binary.BigEndian.Uint16(header[2:4])
		length := binary.BigEndian.Uint16(header[4:6])
		unitID := header[6]
		if protocolID != 0 {
			return fmt.Errorf("modbusserver: unsupported protocol id %d", protocolID)
		}
		if length < 2 || length > 254 {
			return fmt.Errorf("modbusserver: invalid MBAP length %d", length)
		}

		pdu := make([]byte, int(length)-1)
		if _, err := io.ReadFull(conn, pdu); err != nil {
			return err
		}
		responsePDU := s.handlePDU(pdu)
		if len(responsePDU) == 0 {
			return fmt.Errorf("modbusserver: empty response PDU")
		}
		if err := writeResponse(conn, transactionID, unitID, responsePDU); err != nil {
			return err
		}
	}
}

func writeResponse(conn net.Conn, transactionID uint16, unitID byte, pdu []byte) error {
	packet := make([]byte, 7+len(pdu))
	binary.BigEndian.PutUint16(packet[0:2], transactionID)
	binary.BigEndian.PutUint16(packet[2:4], 0)
	binary.BigEndian.PutUint16(packet[4:6], uint16(len(pdu)+1))
	packet[6] = unitID
	copy(packet[7:], pdu)

	for len(packet) > 0 {
		n, err := conn.Write(packet)
		if err != nil {
			return err
		}
		packet = packet[n:]
	}
	return nil
}
