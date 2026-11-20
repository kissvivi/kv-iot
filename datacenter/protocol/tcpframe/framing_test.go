package tcpframe

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestLengthFramingRoundTrip(t *testing.T) {
	framing := Framing{
		Type:         FramingLength,
		LengthOffset: 0,
		LengthSize:   4,
		Endian:       "big",
	}
	payload := []byte{0xAA, 0x55, 0x01}
	frame, err := framing.encodePayload(payload, 1024)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	_ = client.SetDeadline(time.Now().Add(time.Second))

	go func() {
		_, _ = server.Write(frame)
	}()
	decoded, err := framing.readFrame(client, 1024)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(decoded, frame) {
		t.Fatalf("frame = %x, want %x", decoded, frame)
	}
}

func TestFixedFraming(t *testing.T) {
	framing := Framing{Type: FramingFixed, Length: 4}
	payload := []byte{1, 2, 3, 4}
	frame, err := framing.encodePayload(payload, 16)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	_ = client.SetDeadline(time.Now().Add(time.Second))

	go func() {
		_, _ = server.Write(frame)
	}()
	decoded, err := framing.readFrame(client, 16)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatalf("frame = %x, want %x", decoded, payload)
	}
	if _, err := framing.encodePayload([]byte{1}, 16); err == nil {
		t.Fatal("fixed framing accepted a short payload")
	}
}

func TestDelimiterFraming(t *testing.T) {
	framing := Framing{Type: FramingDelimiter, Delimiter: []byte("\r\n")}
	frame, err := framing.encodePayload([]byte("hello"), 64)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if string(frame) != "hello\r\n" {
		t.Fatalf("frame = %q", frame)
	}

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	_ = client.SetDeadline(time.Now().Add(time.Second))
	go func() {
		_, _ = server.Write(frame)
	}()
	decoded, err := framing.readFrame(client, 64)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(decoded) != "hello\r\n" {
		t.Fatalf("decoded = %q", decoded)
	}
}

func TestLengthFramingIncludesHeader(t *testing.T) {
	framing := Framing{
		Type:                 FramingLength,
		LengthOffset:         1,
		LengthSize:           2,
		Endian:               "little",
		LengthIncludesHeader: true,
	}
	frame, err := framing.encodePayload([]byte{0x55, 0x66}, 64)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if frame[0] != 0 || frame[1] != 5 || frame[2] != 0 {
		t.Fatalf("header = %x", frame[:3])
	}
}
