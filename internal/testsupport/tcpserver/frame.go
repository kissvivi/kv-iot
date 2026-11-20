package tcpserver

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

// DefaultMaxFrameSize bounds length-prefixed frames in tests.
const DefaultMaxFrameSize = 1 << 20

// FrameConfig controls a length-prefixed frame server.
type FrameConfig struct {
	Addr         string
	MaxFrameSize uint32
	Respond      func(ctx context.Context, payload []byte) ([]byte, error)
}

// NewLengthPrefixedEcho creates a frame server whose response payload equals
// the request payload. Frames use a big-endian uint32 length prefix.
func NewLengthPrefixedEcho(cfg FrameConfig) *Server {
	if cfg.Respond == nil {
		cfg.Respond = func(_ context.Context, payload []byte) ([]byte, error) {
			return append([]byte(nil), payload...), nil
		}
	}
	maxFrameSize := cfg.MaxFrameSize
	if maxFrameSize == 0 {
		maxFrameSize = DefaultMaxFrameSize
	}

	handler := func(ctx context.Context, conn net.Conn) error {
		for {
			payload, err := ReadFrame(conn, maxFrameSize)
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
					return nil
				}
				return err
			}
			response, err := cfg.Respond(ctx, payload)
			if err != nil {
				return err
			}
			if uint32(len(response)) > maxFrameSize {
				return fmt.Errorf("tcpserver: response frame exceeds %d bytes", maxFrameSize)
			}
			if err := WriteFrame(conn, response); err != nil {
				return err
			}
		}
	}
	return NewWithConfig(Config{Addr: cfg.Addr}, handler)
}

// ReadFrame reads one big-endian uint32 length-prefixed payload.
func ReadFrame(r io.Reader, maxFrameSize uint32) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(header[:])
	if maxFrameSize > 0 && size > maxFrameSize {
		return nil, fmt.Errorf("tcpserver: frame size %d exceeds limit %d", size, maxFrameSize)
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// WriteFrame writes one big-endian uint32 length-prefixed payload.
func WriteFrame(w io.Writer, payload []byte) error {
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}
