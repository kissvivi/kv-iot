package tcpserver

import (
	"context"
	"io"
	"net"
)

// NewEcho creates a TCP server that copies every received byte back to the
// client.
func NewEcho() *Server {
	return New(func(_ context.Context, conn net.Conn) error {
		_, err := io.Copy(conn, conn)
		return err
	})
}

// NewEchoWithConfig creates an echo server with an explicit address.
func NewEchoWithConfig(cfg Config) *Server {
	return NewWithConfig(cfg, func(_ context.Context, conn net.Conn) error {
		_, err := io.Copy(conn, conn)
		return err
	})
}
