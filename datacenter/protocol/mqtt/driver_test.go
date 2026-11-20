package mqtt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
	"kv-iot/internal/testsupport/mqttbroker"
)

var (
	_ protocol.ProtocolDriver = (*Driver)(nil)
	_ protocol.StreamDriver   = (*Driver)(nil)
)

func TestDriverUplinkJSONAndRaw(t *testing.T) {
	broker := startTestBroker(t)
	driverConfig := testDriverConfig(t, broker.Addr(), []map[string]any{
		{"topic": "iot/+/+/uplink", "qos": 1},
		{"topic": "legacy/+/raw", "qos": 1},
	})
	driver := connectTestDriver(t, driverConfig)

	jsonPoint := testPoint(t, "temperature", "float64", `{
		"direction":"uplink",
		"topicTemplate":"iot/{productKey}/{deviceNo}/uplink",
		"identity":{"productKey":"topic.1","deviceNo":"topic.2"},
		"payloadMode":"json_path",
		"valuePath":"data.temperature"
	}`)
	rawPoint := testPoint(t, "rawPayload", "string", `{
		"direction":"uplink",
		"topicTemplate":"legacy/{deviceNo}/raw",
		"identity":{"deviceNo":"topic.1"},
		"payloadMode":"raw",
		"raw":{"encoding":"utf8"}
	}`)

	if err := broker.Publish("iot/p1/d1/uplink", []byte(`{"data":{"temperature":23.5}}`), false); err != nil {
		t.Fatalf("publish json: %v", err)
	}
	if err := broker.Publish("legacy/d1/raw", []byte("online"), false); err != nil {
		t.Fatalf("publish raw: %v", err)
	}

	samples := waitSamples(t, driver, []protocol.PointBinding{jsonPoint, rawPoint})
	values := make(map[string]any, len(samples))
	for _, sample := range samples {
		values[sample.PointIdentifier] = sample.Value
	}
	if values["temperature"] != 23.5 {
		t.Fatalf("temperature = %#v", values["temperature"])
	}
	if values["rawPayload"] != "online" {
		t.Fatalf("rawPayload = %#v", values["rawPayload"])
	}
}

func TestDriverDownlinkPublish(t *testing.T) {
	broker := startTestBroker(t)
	driver := connectTestDriver(t, testDriverConfig(t, broker.Addr(), nil))
	point := testPoint(t, "switch", "", `{
		"direction":"downlink",
		"topicTemplate":"iot/{productKey}/{deviceNo}/downlink",
		"qos":1,
		"payloadTemplate":{
			"action":"${actionIdentifier}",
			"params":"${params}",
			"requestId":"${requestId}"
		}
	}`)

	err := driver.Write(context.Background(), point, map[string]any{
		"requestId": "req-1",
		"params":    map[string]any{"enabled": true},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	message := waitPublished(t, broker)
	if message.Topic != "iot/p1/d1/downlink" {
		t.Fatalf("topic = %q", message.Topic)
	}
	var payload map[string]any
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["action"] != "switch" || payload["requestId"] != "req-1" {
		t.Fatalf("payload = %#v", payload)
	}
	params, ok := payload["params"].(map[string]any)
	if !ok || params["enabled"] != true {
		t.Fatalf("params = %#v", payload["params"])
	}
}

func TestDriverDownlinkAck(t *testing.T) {
	broker := startTestBroker(t)
	driver := connectTestDriver(t, testDriverConfig(t, broker.Addr(), nil))
	point := testPoint(t, "switch", "", `{
		"direction":"downlink",
		"topicTemplate":"iot/{productKey}/{deviceNo}/downlink",
		"qos":1,
		"payloadTemplate":{"requestId":"${requestId}","params":"${params}"},
		"response":{
			"topicTemplate":"iot/{productKey}/{deviceNo}/ack",
			"correlationField":"requestId",
			"timeoutMs":2000
		}
	}`)

	result := make(chan error, 1)
	go func() {
		result <- driver.Write(context.Background(), point, map[string]any{"enabled": true})
	}()

	outbound := waitPublished(t, broker)
	var payload map[string]any
	if err := json.Unmarshal(outbound.Payload, &payload); err != nil {
		t.Fatalf("decode outbound payload: %v", err)
	}
	requestID, _ := payload["requestId"].(string)
	if requestID == "" {
		t.Fatalf("requestId missing from payload: %s", outbound.Payload)
	}
	ack, _ := json.Marshal(map[string]any{"requestId": requestID, "ok": true})
	if err := broker.Publish("iot/p1/d1/ack", ack, false); err != nil {
		t.Fatalf("publish ack: %v", err)
	}

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for Write")
	}
}

func TestDriverDownlinkAckTimeout(t *testing.T) {
	broker := startTestBroker(t)
	driver := connectTestDriver(t, testDriverConfig(t, broker.Addr(), nil))
	point := testPoint(t, "switch", "", `{
		"direction":"downlink",
		"topicTemplate":"iot/{productKey}/{deviceNo}/downlink",
		"qos":1,
		"payloadTemplate":{"requestId":"${requestId}"},
		"response":{
			"topicTemplate":"iot/{productKey}/{deviceNo}/ack",
			"correlationField":"requestId",
			"timeoutMs":100
		}
	}`)

	err := driver.Write(context.Background(), point, map[string]any{})
	if !errors.Is(err, ErrAckTimeout) {
		t.Fatalf("error = %v, want ErrAckTimeout", err)
	}
}

func TestDriverRestoresSubscriptionsAfterReconnect(t *testing.T) {
	broker := startTestBroker(t)
	driverConfig := testDriverConfig(t, broker.Addr(), []map[string]any{
		{"topic": "iot/+/+/uplink", "qos": 1},
	})
	driver := connectTestDriver(t, driverConfig)
	waitFor(t, 2*time.Second, func() bool {
		return broker.SubscriptionCount() == 1
	}, "initial subscription")

	if err := driver.Close(); err != nil {
		t.Fatalf("close driver: %v", err)
	}
	waitFor(t, 2*time.Second, func() bool {
		return broker.SubscriptionCount() == 0
	}, "subscription cleanup")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := driver.Connect(ctx, driverConfig); err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	waitFor(t, 2*time.Second, func() bool {
		return broker.SubscriptionCount() == 1
	}, "restored subscription")
}

func TestDriverSubscribeMessages(t *testing.T) {
	broker := startTestBroker(t)
	driver := connectTestDriver(t, testDriverConfig(t, broker.Addr(), []map[string]any{
		{"topic": "raw/#", "qos": 1},
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	received := make(chan Message, 1)
	go func() {
		err := driver.SubscribeMessages(ctx, func(message Message) {
			received <- message
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("SubscribeMessages: %v", err)
		}
	}()

	if err := broker.Publish("raw/device-1", []byte("hello"), false); err != nil {
		t.Fatalf("publish: %v", err)
	}
	select {
	case message := <-received:
		if message.Topic != "raw/device-1" || string(message.Payload) != "hello" {
			t.Fatalf("message = %#v", message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stream message")
	}
}

func startTestBroker(t *testing.T) *mqttbroker.Broker {
	t.Helper()
	broker := mqttbroker.New(mqttbroker.Config{})
	if err := broker.Start(); err != nil {
		t.Fatalf("start broker: %v", err)
	}
	t.Cleanup(func() {
		if err := broker.Close(); err != nil {
			t.Errorf("close broker: %v", err)
		}
	})
	return broker
}

func testDriverConfig(t *testing.T, address string, subscriptions []map[string]any) protocol.DriverConfig {
	t.Helper()
	config := map[string]any{
		"broker":            "tcp://" + address,
		"clientId":          fmt.Sprintf("test-%d", time.Now().UnixNano()),
		"connectTimeoutMs":  2000,
		"writeTimeoutMs":    2000,
		"defaultPublishQoS": 1,
	}
	if len(subscriptions) > 0 {
		config["subscriptions"] = subscriptions
	}
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal driver config: %v", err)
	}
	return protocol.DriverConfig{
		ChannelID:    "mqtt-test",
		ProtocolType: protocol.ProtocolMQTT,
		Config:       raw,
	}
}

func connectTestDriver(t *testing.T, config protocol.DriverConfig) *Driver {
	t.Helper()
	driver, err := New(config)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := driver.Connect(ctx, config); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() {
		if err := driver.Close(); err != nil {
			t.Errorf("close driver: %v", err)
		}
	})
	return driver
}

func testPoint(t *testing.T, identifier, dataType, rawConfig string) protocol.PointBinding {
	t.Helper()
	return protocol.PointBinding{
		PointID:         int64(len(identifier)),
		ProductID:       1,
		ProductKey:      "p1",
		DeviceID:        2,
		DeviceNo:        "d1",
		PointIdentifier: identifier,
		PointKind:       "property",
		DataType:        dataType,
		Config:          json.RawMessage(rawConfig),
	}
}

func waitSamples(t *testing.T, driver *Driver, points []protocol.PointBinding) []protocol.PointSample {
	t.Helper()
	var samples []protocol.PointSample
	waitFor(t, 3*time.Second, func() bool {
		result, err := driver.Read(context.Background(), points)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if len(result) > 0 {
			samples = append(samples, result...)
		}
		return len(samples) >= len(points)
	}, "uplink samples")
	return samples
}

func waitPublished(t *testing.T, broker *mqttbroker.Broker) mqttbroker.Message {
	t.Helper()
	select {
	case message := <-broker.Published():
		return message
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for publication")
		return mqttbroker.Message{}
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool, label string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", strings.TrimSpace(label))
}
