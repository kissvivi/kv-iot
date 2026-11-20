package tcpframe

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
	"kv-iot/internal/testsupport/tcpserver"
)

func TestReadWithLengthFraming(t *testing.T) {
	server := tcpserver.NewLengthPrefixedEcho(tcpserver.FrameConfig{
		Respond: func(_ context.Context, payload []byte) ([]byte, error) {
			return []byte{0xAA, 0x55, 0x00, 0x7B}, nil
		},
	})
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer server.Close()

	driver := newFrameDriver(t, server.Addr(), `{}`)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := driver.Connect(ctx, frameDriverConfig(server.Addr(), `{}`)); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer driver.Close()

	point := protocol.PointBinding{
		PointID:         7,
		DeviceID:        9,
		ProductID:       3,
		PointIdentifier: "temperature",
		Config: json.RawMessage(`{
			"requestHex": "0102",
			"response": {
				"matchPrefixHex": "00000004AA55",
				"fields": [
					{"identifier": "temperature", "offset": 6, "type": "uint16", "scale": 0.1}
				]
			}
		}`),
	}
	samples, err := driver.Read(ctx, []protocol.PointBinding{point})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("samples = %d", len(samples))
	}
	value, ok := samples[0].Value.(float64)
	if !ok || math.Abs(value-12.3) > 1e-9 {
		t.Fatalf("value = %#v", samples[0].Value)
	}
	if samples[0].PointIdentifier != "temperature" || samples[0].PointID != 7 {
		t.Fatalf("sample metadata = %#v", samples[0])
	}
	if driver.Health().Status != protocol.HealthHealthy {
		t.Fatalf("health = %s", driver.Health().Status)
	}
}

func TestWritePublishesFramedRequest(t *testing.T) {
	received := make(chan []byte, 1)
	server := tcpserver.NewLengthPrefixedEcho(tcpserver.FrameConfig{
		Respond: func(_ context.Context, payload []byte) ([]byte, error) {
			select {
			case received <- append([]byte(nil), payload...):
			default:
			}
			return []byte{0x06}, nil
		},
	})
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer server.Close()

	driver := newFrameDriver(t, server.Addr(), `{}`)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := driver.Connect(ctx, frameDriverConfig(server.Addr(), `{}`)); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer driver.Close()

	point := protocol.PointBinding{
		PointIdentifier: "unlock",
		DataType:        "uint16",
		Config: json.RawMessage(`{
			"requestTemplate": "AA55{value:u16le}",
			"expectResponse": false
		}`),
	}
	if err := driver.Write(ctx, point, uint16(0x1234)); err != nil {
		t.Fatalf("Write: %v", err)
	}

	select {
	case payload := <-received:
		want := []byte{0xAA, 0x55, 0x34, 0x12}
		if string(payload) != string(want) {
			t.Fatalf("payload = %x, want %x", payload, want)
		}
	case <-ctx.Done():
		t.Fatal("server did not receive request")
	}
}

func TestFrameReconnectOnNextOperation(t *testing.T) {
	var connections atomic.Int32
	server := tcpserver.New(func(_ context.Context, conn net.Conn) error {
		connection := connections.Add(1)
		payload, err := tcpserver.ReadFrame(conn, 1024)
		if err != nil {
			return err
		}
		if connection == 1 {
			return nil
		}
		_ = payload
		return tcpserver.WriteFrame(conn, []byte{0xAA, 0x55, 0x00, 0x01})
	})
	if err := server.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer server.Close()

	driver := newFrameDriver(t, server.Addr(), `{"reconnectDelay":"1ms","maxReconnectAttempts":2}`)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := driver.Connect(ctx, frameDriverConfig(server.Addr(), `{"reconnectDelay":"1ms","maxReconnectAttempts":2}`)); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer driver.Close()

	point := protocol.PointBinding{
		PointIdentifier: "status",
		Config: json.RawMessage(`{
			"requestHex": "01",
			"fields": [{"identifier": "status", "offset": 7, "type": "uint8"}]
		}`),
	}
	_, _ = driver.Read(ctx, []protocol.PointBinding{point})

	deadline := time.Now().Add(2 * time.Second)
	for {
		samples, err := driver.Read(ctx, []protocol.PointBinding{point})
		if err == nil {
			if len(samples) != 1 || samples[0].Value != uint64(1) {
				t.Fatalf("samples = %#v", samples)
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

func TestParseLengthFramingConfig(t *testing.T) {
	raw := json.RawMessage(`{
		"host": "127.0.0.1",
		"port": 9000,
		"timeoutMs": 500,
		"framing": {
			"type": "length",
			"lengthOffset": 0,
			"lengthSize": 4,
			"endian": "big"
		}
	}`)
	cfg, err := parseConfig(raw)
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if cfg.Framing.Type != FramingLength || cfg.Framing.LengthSize != 4 {
		t.Fatalf("framing = %#v", cfg.Framing)
	}
}

func newFrameDriver(t *testing.T, address, options string) *Driver {
	t.Helper()
	driver, err := New(frameDriverConfig(address, options))
	if err != nil {
		t.Fatalf("new driver: %v", err)
	}
	return driver
}

func frameDriverConfig(address, options string) protocol.DriverConfig {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		panic(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		panic(err)
	}
	var config map[string]any
	if err := json.Unmarshal([]byte(options), &config); err != nil {
		panic(err)
	}
	config["host"] = host
	config["port"] = port
	config["connectTimeout"] = "1s"
	config["readTimeout"] = "1s"
	config["writeTimeout"] = "1s"
	if _, ok := config["framing"]; !ok {
		config["framing"] = map[string]any{
			"type":         "length",
			"lengthOffset": 0,
			"lengthSize":   4,
			"endian":       "big",
		}
	}
	raw, err := json.Marshal(config)
	if err != nil {
		panic(fmt.Errorf("marshal config: %w", err))
	}
	return protocol.DriverConfig{
		ChannelID:    "tcpframe-test",
		ProtocolType: protocol.ProtocolTCPFrame,
		Config:       raw,
	}
}
