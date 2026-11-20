package mqtt

import (
	"encoding/json"
	"errors"
	"testing"

	"kv-iot/datacenter/protocol"
)

func TestParseConfigDefaultsAndQoSZero(t *testing.T) {
	zero := 0
	raw, err := json.Marshal(map[string]any{
		"broker":            "127.0.0.1:1883",
		"defaultPublishQoS": zero,
		"subscriptions": []map[string]any{
			{"topic": "iot/+/+/uplink", "qos": zero},
		},
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	config, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if config.Broker != "tcp://127.0.0.1:1883" {
		t.Fatalf("broker = %q", config.Broker)
	}
	if config.publishQoS() != 0 {
		t.Fatalf("publish qos = %d, want 0", config.publishQoS())
	}
	if config.subscribeQoS() != defaultSubscribeQoS {
		t.Fatalf("subscribe qos = %d", config.subscribeQoS())
	}
	if got := config.Subscriptions[0].valueQoS(config.subscribeQoS()); got != 0 {
		t.Fatalf("subscription qos = %d, want 0", got)
	}
	if config.InboundQueueSize != defaultInboundQueueSize {
		t.Fatalf("inbound queue = %d", config.InboundQueueSize)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want error
	}{
		{
			name: "valid",
			raw:  `{"broker":"tcp://127.0.0.1:1883","subscriptions":[{"topic":"devices/+/state","qos":2}]}`,
		},
		{
			name: "missing broker",
			raw:  `{}`,
			want: ErrInvalidBroker,
		},
		{
			name: "unsupported scheme",
			raw:  `{"broker":"udp://127.0.0.1:1883"}`,
			want: ErrInvalidBroker,
		},
		{
			name: "invalid qos",
			raw:  `{"broker":"tcp://127.0.0.1:1883","defaultPublishQoS":3}`,
			want: ErrInvalidQoS,
		},
		{
			name: "invalid filter",
			raw:  `{"broker":"tcp://127.0.0.1:1883","subscriptions":[{"topic":"a/#/b"}]}`,
			want: protocol.ErrInvalidConfig,
		},
		{
			name: "certificate without key",
			raw:  `{"broker":"tls://127.0.0.1:8883","tls":{"enabled":true,"certFile":"client.crt"}}`,
			want: protocol.ErrInvalidConfig,
		},
		{
			name: "unknown field",
			raw:  `{"broker":"tcp://127.0.0.1:1883","unknown":true}`,
			want: protocol.ErrInvalidConfig,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateConfig(json.RawMessage(test.raw))
			if test.want == nil {
				if err != nil {
					t.Fatalf("ValidateConfig: %v", err)
				}
				return
			}
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestRegistryCompatibleFactory(t *testing.T) {
	registry := protocol.NewRegistry()
	if err := registry.RegisterWithValidator(protocol.ProtocolMQTT, NewDriver, Validator); err != nil {
		t.Fatalf("register mqtt: %v", err)
	}
	driver, err := registry.Create(protocol.ProtocolMQTT, protocol.DriverConfig{
		ChannelID:    "mqtt-main",
		ProtocolType: protocol.ProtocolMQTT,
		Config:       json.RawMessage(`{"broker":"tcp://127.0.0.1:1883"}`),
	})
	if err != nil {
		t.Fatalf("create mqtt driver: %v", err)
	}
	if driver == nil {
		t.Fatal("driver is nil")
	}
	if err := driver.Close(); err != nil {
		t.Fatalf("close driver: %v", err)
	}
}
