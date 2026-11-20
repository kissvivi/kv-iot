package mqtt

import (
	"context"
	"testing"
	"time"

	"kv-iot/internal/testsupport/mqttbroker"
)

func TestMqtt_Pub(t *testing.T) {
	broker := mqttbroker.New(mqttbroker.Config{})
	if err := broker.Start(); err != nil {
		t.Fatalf("start broker: %v", err)
	}
	t.Cleanup(func() {
		_ = broker.Close()
	})

	host, port := splitTestBrokerAddr(t, broker.Addr())
	mc := NewMqtt(NewConfigMqtt(host, port, "mqtt-pub-test", "", ""))
	if mc == nil {
		t.Fatal("mqtt client was not created")
	}
	t.Cleanup(func() {
		mc.Conn.Disconnect(100)
	})

	mc.Pub(1, "iot/test/device/uplink", `{"value":25.5}`)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		select {
		case message := <-broker.Published():
			if message.Topic == "iot/test/device/uplink" {
				return
			}
		case <-ctx.Done():
			t.Fatal("published message was not observed by broker")
		}
	}
}
