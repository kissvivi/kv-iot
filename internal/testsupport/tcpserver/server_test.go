package tcpserver

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"
)

func TestEchoServer(t *testing.T) {
	server := NewEcho()
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("close server: %v", err)
		}
	})

	conn := dialTestServer(t, server.Addr())
	defer conn.Close()

	payload := []byte("hello tcp")
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("echo mismatch: got %q want %q", got, payload)
	}
}

func TestLengthPrefixedEcho(t *testing.T) {
	server := NewLengthPrefixedEcho(FrameConfig{
		MaxFrameSize: 1024,
		Respond: func(_ context.Context, payload []byte) ([]byte, error) {
			return append([]byte("reply:"), payload...), nil
		},
	})
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("close server: %v", err)
		}
	})

	conn := dialTestServer(t, server.Addr())
	defer conn.Close()

	if err := WriteFrame(conn, []byte("frame")); err != nil {
		t.Fatalf("write frame: %v", err)
	}
	got, err := ReadFrame(conn, 1024)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if string(got) != "reply:frame" {
		t.Fatalf("frame response = %q", got)
	}
}

func TestServerCloseIsIdempotent(t *testing.T) {
	server := NewEcho()
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	if err := server.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := server.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	if err := server.Start(); err != ErrClosed {
		t.Fatalf("start after close = %v, want %v", err, ErrClosed)
	}
}

func dialTestServer(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	return conn
}
