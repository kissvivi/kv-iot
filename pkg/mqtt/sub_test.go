package mqtt

import (
	"net"
	"strconv"
	"testing"
	"time"

	"kv-iot/internal/testsupport/mqttbroker"
)

func TestMqtt_Sub(t *testing.T) {
	broker := mqttbroker.New(mqttbroker.Config{})
	if err := broker.Start(); err != nil {
		t.Fatalf("start broker: %v", err)
	}
	t.Cleanup(func() {
		_ = broker.Close()
	})

	host, port := splitTestBrokerAddr(t, broker.Addr())
	mc := NewMqtt(NewConfigMqtt(host, port, "mqtt-sub-test", "", ""))
	if mc == nil {
		t.Fatal("mqtt client was not created")
	}
	t.Cleanup(func() {
		mc.Conn.Disconnect(100)
	})

	mc.Sub(1, "iot/+/+/uplink")
	deadline := time.Now().Add(2 * time.Second)
	for broker.SubscriptionCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if broker.SubscriptionCount() == 0 {
		t.Fatal("subscription was not registered")
	}

	if err := broker.Publish("iot/test/device/uplink", []byte(`{"value":1}`), false); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case message := <-SubData:
		if message == nil {
			t.Fatal("received nil message")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subscribed message")
	}
}

func splitTestBrokerAddr(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split broker address %q: %v", addr, err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parse broker port %q: %v", portText, err)
	}
	return host, port
}
