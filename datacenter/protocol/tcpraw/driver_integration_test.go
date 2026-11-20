package tcpraw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
	"kv-iot/internal/testsupport/tcpserver"
)

func TestSendRawRequestResponse(t *testing.T) {
	server := tcpserver.New(func(_ context.Context, conn net.Conn) error {
		buffer := make([]byte, 32)
		for {
			count, err := conn.Read(buffer)
			if count > 0 {
				if _, writeErr := conn.Write(append([]byte("response:"), buffer[:count]...)); writeErr != nil {
					return writeErr
				}
			}
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
		}
	})
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer server.Close()

	driver := newTestDriver(t, server.Addr(), `{}`)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := driver.Connect(ctx, driverConfig(server.Addr(), `{}`)); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer driver.Close()

	response, err := driver.SendRaw(ctx, []byte("ping"))
	if err != nil {
		t.Fatalf("SendRaw: %v", err)
	}
	if string(response) != "response:ping" {
		t.Fatalf("response = %q", response)
	}
	if driver.Health().Status != protocol.HealthHealthy {
		t.Fatalf("health = %s", driver.Health().Status)
	}
}

func TestSubscribeByteStream(t *testing.T) {
	start := make(chan struct{})
	server := tcpserver.New(func(_ context.Context, conn net.Conn) error {
		<-start
		_, err := conn.Write([]byte("unsolicited"))
		return err
	})
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer server.Close()

	driver := newTestDriver(t, server.Addr(), `{}`)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := driver.Connect(ctx, driverConfig(server.Addr(), `{}`)); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer driver.Close()

	received := make(chan []byte, 1)
	subscribeDone := make(chan error, 1)
	go func() {
		subscribeDone <- driver.Subscribe(ctx, func(payload []byte) {
			select {
			case received <- payload:
			default:
			}
		})
	}()
	time.Sleep(50 * time.Millisecond)
	close(start)

	select {
	case payload := <-received:
		if string(payload) != "unsolicited" {
			t.Fatalf("payload = %q", payload)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for stream payload")
	}
	cancel()
	if err := <-subscribeDone; err != nil && err != context.Canceled {
		t.Fatalf("Subscribe: %v", err)
	}
}

func TestReadReturnsResponseBytesAsSample(t *testing.T) {
	server := tcpserver.New(func(_ context.Context, conn net.Conn) error {
		buffer := make([]byte, 8)
		count, err := conn.Read(buffer)
		if count > 0 {
			_, _ = conn.Write(buffer[:count])
		}
		return err
	})
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer server.Close()

	driver := newTestDriver(t, server.Addr(), `{}`)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := driver.Connect(ctx, driverConfig(server.Addr(), `{}`)); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer driver.Close()

	samples, err := driver.Read(ctx, []protocol.PointBinding{{
		PointIdentifier: "raw",
		Config:          json.RawMessage(`{"requestHex":"0102"}`),
	}})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("samples = %d", len(samples))
	}
	value, ok := samples[0].Value.([]byte)
	if !ok || !bytes.Equal(value, []byte{1, 2}) {
		t.Fatalf("sample value = %#v", samples[0].Value)
	}
}

func TestReconnectOnNextOperation(t *testing.T) {
	var connections atomic.Int32
	server := tcpserver.New(func(_ context.Context, conn net.Conn) error {
		attempt := connections.Add(1)
		buffer := make([]byte, 16)
		_, err := conn.Read(buffer)
		if err != nil {
			return err
		}
		if attempt == 1 {
			return nil
		}
		_, err = conn.Write([]byte("reconnected"))
		return err
	})
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer server.Close()

	driver := newTestDriver(t, server.Addr(), `{"reconnectDelay":"1ms","maxReconnectAttempts":2}`)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := driver.Connect(ctx, driverConfig(server.Addr(), `{"reconnectDelay":"1ms","maxReconnectAttempts":2}`)); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer driver.Close()

	_, _ = driver.SendRaw(ctx, []byte("first"))
	deadline := time.Now().Add(2 * time.Second)
	for {
		response, err := driver.SendRaw(ctx, []byte("second"))
		if err == nil {
			if string(response) != "reconnected" {
				t.Fatalf("response = %q", response)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("reconnect did not succeed: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if connections.Load() < 2 {
		t.Fatalf("connections = %d, want at least 2", connections.Load())
	}
}

func TestResponseModes(t *testing.T) {
	fixed := Config{ResponseMode: ResponseModeFixed, ResponseLength: 3}
	if complete, value := responseComplete([]byte{1, 2}, fixed); complete {
		t.Fatal("fixed response completed early")
	} else if value != nil {
		t.Fatalf("fixed partial value = %v", value)
	}
	if complete, value := responseComplete([]byte{1, 2, 3, 4}, fixed); !complete || !bytes.Equal(value, []byte{1, 2, 3}) {
		t.Fatalf("fixed result = %v, %v", complete, value)
	}

	delimiter := Config{ResponseMode: ResponseModeDelimiter, ResponseDelimiter: []byte("\r\n")}
	if complete, value := responseComplete([]byte("abc\r"), delimiter); complete {
		t.Fatal("delimiter response completed early")
	} else if value != nil {
		t.Fatalf("delimiter partial value = %v", value)
	}
	if complete, value := responseComplete([]byte("abc\r\nnext"), delimiter); !complete || string(value) != "abc\r\n" {
		t.Fatalf("delimiter result = %v, %q", complete, value)
	}
}

func newTestDriver(t *testing.T, address, options string) *Driver {
	t.Helper()
	driver, err := New(driverConfig(address, options))
	if err != nil {
		t.Fatalf("new driver: %v", err)
	}
	return driver
}

func driverConfig(address, options string) protocol.DriverConfig {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		panic(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		panic(err)
	}
	var extra map[string]any
	if err := json.Unmarshal([]byte(options), &extra); err != nil {
		panic(err)
	}
	extra["host"] = host
	extra["port"] = port
	extra["connectTimeout"] = "1s"
	extra["readTimeout"] = "1s"
	extra["writeTimeout"] = "1s"
	if _, ok := extra["responseTimeout"]; !ok {
		extra["responseTimeout"] = "1s"
	}
	raw, err := json.Marshal(extra)
	if err != nil {
		panic(fmt.Errorf("marshal config: %w", err))
	}
	return protocol.DriverConfig{
		ChannelID:    "tcpraw-test",
		ProtocolType: protocol.ProtocolTCPRaw,
		Config:       raw,
	}
}
