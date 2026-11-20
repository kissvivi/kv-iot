package mqtt

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
)

func TestDecodeUplinkJSONPath(t *testing.T) {
	config := mustPointConfig(t, `{
		"direction":"uplink",
		"topicTemplate":"iot/{productKey}/{deviceNo}/uplink",
		"identity":{"productKey":"topic.1","deviceNo":"topic.2"},
		"payloadMode":"json_path",
		"valuePath":"data.temperature",
		"timestampPath":"data.time"
	}`)
	point := protocol.PointBinding{
		ProductID:       7,
		ProductKey:      "p1",
		DeviceID:        9,
		DeviceNo:        "d1",
		PointID:         11,
		PointIdentifier: "temperature",
		DataType:        "float64",
	}
	message := inboundMessage{
		Topic:      "iot/p1/d1/uplink",
		Payload:    []byte(`{"data":{"temperature":"21.5","time":"2026-11-19T10:00:00Z"}}`),
		ReceivedAt: time.Now(),
	}

	sample, err := decodeUplink(point, config, message)
	if err != nil {
		t.Fatalf("decodeUplink: %v", err)
	}
	if sample.Value != 21.5 {
		t.Fatalf("value = %#v", sample.Value)
	}
	if sample.Timestamp.Format(time.RFC3339) != "2026-11-19T10:00:00Z" {
		t.Fatalf("timestamp = %s", sample.Timestamp)
	}
	if sample.Quality != protocol.QualityGood {
		t.Fatalf("quality = %q", sample.Quality)
	}
}

func TestDecodeUplinkRawEncodings(t *testing.T) {
	point := protocol.PointBinding{
		PointIdentifier: "payload",
		DataType:        "string",
	}
	tests := []struct {
		encoding string
		payload  []byte
		want     string
	}{
		{encoding: "utf8", payload: []byte("hello"), want: "hello"},
		{encoding: "hex", payload: []byte{0x01, 0xff}, want: "01ff"},
		{encoding: "base64", payload: []byte("hello"), want: "aGVsbG8="},
	}
	for _, test := range tests {
		t.Run(test.encoding, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{
				"topicTemplate": "legacy/{deviceNo}/raw",
				"payloadMode":   "raw",
				"raw":           map[string]any{"encoding": test.encoding},
			})
			if err != nil {
				t.Fatalf("marshal config: %v", err)
			}
			config, err := ParsePointConfig(raw)
			if err != nil {
				t.Fatalf("ParsePointConfig: %v", err)
			}
			sample, err := decodeUplink(point, config, inboundMessage{
				Topic:      "legacy/d1/raw",
				Payload:    test.payload,
				ReceivedAt: time.Now(),
			})
			if err != nil {
				t.Fatalf("decodeUplink: %v", err)
			}
			if sample.Value != test.want {
				t.Fatalf("value = %#v, want %q", sample.Value, test.want)
			}
		})
	}
}

func TestDecodeUplinkIdentityMismatch(t *testing.T) {
	config := mustPointConfig(t, `{
		"topicTemplate":"iot/{productKey}/{deviceNo}/uplink",
		"identity":{"productKey":"topic.1","deviceNo":"topic.2"},
		"payloadMode":"json_path",
		"valuePath":"value"
	}`)
	point := protocol.PointBinding{
		ProductKey:      "other",
		DeviceNo:        "d1",
		PointIdentifier: "value",
	}
	_, err := decodeUplink(point, config, inboundMessage{
		Topic:      "iot/p1/d1/uplink",
		Payload:    []byte(`{"value":1}`),
		ReceivedAt: time.Now(),
	})
	if !errors.Is(err, ErrInvalidBinding) {
		t.Fatalf("error = %v, want ErrInvalidBinding", err)
	}
}

func mustPointConfig(t *testing.T, raw string) PointConfig {
	t.Helper()
	config, err := ParsePointConfig(json.RawMessage(raw))
	if err != nil {
		t.Fatalf("ParsePointConfig: %v", err)
	}
	return config
}
