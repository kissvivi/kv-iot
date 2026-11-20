package integration_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"kv-iot/datacenter/protocol"
	mqttdriver "kv-iot/datacenter/protocol/mqtt"
	"kv-iot/internal/testsupport/mqttbroker"
)

func TestMQTTDriverUplinkDownlinkAndACK(t *testing.T) {
	broker := mqttbroker.New(mqttbroker.Config{})
	if err := broker.Start(); err != nil {
		t.Fatalf("start MQTT broker: %v", err)
	}
	t.Cleanup(func() {
		if err := broker.Close(); err != nil {
			t.Errorf("close MQTT broker: %v", err)
		}
	})

	registry := protocol.NewRegistry()
	if err := registry.RegisterWithValidator(protocol.ProtocolMQTT, mqttdriver.NewDriver, mqttdriver.Validator); err != nil {
		t.Fatalf("register MQTT driver: %v", err)
	}

	driverConfig := protocol.DriverConfig{
		ChannelID:    "mqtt-integration",
		ProtocolType: protocol.ProtocolMQTT,
		Config: mustJSON(t, map[string]any{
			"broker":           "tcp://" + broker.Addr(),
			"clientId":         "mqtt-integration-client",
			"connectTimeoutMs": 2000,
			"writeTimeoutMs":   2000,
			"subscriptions": []map[string]any{
				{"topic": "iot/+/+/uplink", "qos": 1},
				{"topic": "iot/+/+/ack", "qos": 1},
			},
		}),
	}
	driver, err := registry.Create(protocol.ProtocolMQTT, driverConfig)
	if err != nil {
		t.Fatalf("create MQTT driver: %v", err)
	}
	t.Cleanup(func() {
		_ = driver.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := driver.Connect(ctx, driverConfig); err != nil {
		t.Fatalf("connect MQTT driver: %v", err)
	}
	waitFor(t, 2*time.Second, "MQTT subscriptions", func() bool {
		return broker.SubscriptionCount() == 2
	})

	uplinkPoint := protocol.PointBinding{
		PointID:         101,
		ProductID:       7,
		ProductKey:      "p1",
		DeviceID:        8,
		DeviceNo:        "d1",
		PointIdentifier: "temperature",
		PointKind:       "property",
		DataType:        "float64",
		Config: mustJSON(t, map[string]any{
			"direction":     "uplink",
			"topicTemplate": "iot/{productKey}/{deviceNo}/uplink",
			"identity": map[string]any{
				"productKey": "topic.1",
				"deviceNo":   "topic.2",
			},
			"payloadMode": "json_path",
			"valuePath":   "data.temperature",
		}),
	}
	if err := broker.Publish("iot/p1/d1/uplink", []byte(`{"data":{"temperature":23.5}}`), false); err != nil {
		t.Fatalf("publish MQTT uplink: %v", err)
	}

	var uplinkSamples []protocol.PointSample
	waitFor(t, 3*time.Second, "MQTT uplink sample", func() bool {
		samples, readErr := driver.Read(context.Background(), []protocol.PointBinding{uplinkPoint})
		if readErr != nil {
			t.Fatalf("read MQTT uplink: %v", readErr)
		}
		uplinkSamples = append(uplinkSamples, samples...)
		return len(uplinkSamples) > 0
	})
	if got := uplinkSamples[0].Value; got != 23.5 {
		t.Fatalf("uplink value = %#v, want 23.5", got)
	}
	if uplinkSamples[0].ChannelID != driverConfig.ChannelID {
		t.Fatalf("uplink channel = %q, want %q", uplinkSamples[0].ChannelID, driverConfig.ChannelID)
	}

	downlinkPoint := protocol.PointBinding{
		PointID:         102,
		ProductID:       7,
		ProductKey:      "p1",
		DeviceID:        8,
		DeviceNo:        "d1",
		PointIdentifier: "switch",
		PointKind:       "action",
		Config: mustJSON(t, map[string]any{
			"direction":     "downlink",
			"topicTemplate": "iot/{productKey}/{deviceNo}/downlink",
			"qos":           1,
			"payloadTemplate": map[string]any{
				"requestId": "${requestId}",
				"params":    "${params}",
			},
			"response": map[string]any{
				"topicTemplate":    "iot/{productKey}/{deviceNo}/ack",
				"correlationField": "requestId",
				"timeoutMs":        2000,
			},
		}),
	}

	writeResult := make(chan error, 1)
	go func() {
		writeResult <- driver.Write(ctx, downlinkPoint, map[string]any{
			"params": map[string]any{"enabled": true},
		})
	}()

	outbound := waitForPublication(t, broker, "iot/p1/d1/downlink", 3*time.Second)
	var command map[string]any
	if err := json.Unmarshal(outbound.Payload, &command); err != nil {
		t.Fatalf("decode MQTT downlink: %v", err)
	}
	requestID, _ := command["requestId"].(string)
	if requestID == "" {
		t.Fatalf("downlink requestId missing: %s", outbound.Payload)
	}
	params, ok := command["params"].(map[string]any)
	if !ok || params["enabled"] != true {
		t.Fatalf("downlink params = %#v", command["params"])
	}

	ackPayload, err := json.Marshal(map[string]any{
		"requestId": requestID,
		"ok":        true,
	})
	if err != nil {
		t.Fatalf("marshal ACK: %v", err)
	}
	if err := broker.Publish("iot/p1/d1/ack", ackPayload, false); err != nil {
		t.Fatalf("publish MQTT ACK: %v", err)
	}
	select {
	case err := <-writeResult:
		if err != nil {
			t.Fatalf("MQTT downlink with ACK: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for MQTT ACK completion")
	}
}

func waitForPublication(t *testing.T, broker *mqttbroker.Broker, topic string, timeout time.Duration) mqttbroker.Message {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case message := <-broker.Published():
			if message.Topic == topic {
				return message
			}
		case <-timer.C:
			t.Fatalf("timed out waiting for MQTT publication on %s", topic)
		}
	}
}
