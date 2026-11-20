package mqttbroker

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestBrokerConnectSubscribePublish(t *testing.T) {
	broker := startBroker(t)

	subscriber := connectClient(t, broker.Addr(), "subscriber")
	defer subscriber.Close()
	subscribe(t, subscriber, 1, "devices/+/status")

	publisher := connectClient(t, broker.Addr(), "publisher")
	defer publisher.Close()
	publish(t, publisher, "devices/a/status", []byte(`{"online":true}`))

	pkt := readPacketWithDeadline(t, subscriber)
	if pkt.header>>4 != packetPublish {
		t.Fatalf("packet type = %d, want PUBLISH", pkt.header>>4)
	}
	offset := 0
	topic, err := readMQTTString(pkt.payload, &offset)
	if err != nil {
		t.Fatalf("read topic: %v", err)
	}
	if topic != "devices/a/status" {
		t.Fatalf("topic = %q", topic)
	}
	if !bytes.Equal(pkt.payload[offset:], []byte(`{"online":true}`)) {
		t.Fatalf("payload = %s", pkt.payload[offset:])
	}

	select {
	case message := <-broker.Published():
		if message.Topic != "devices/a/status" {
			t.Fatalf("event topic = %q", message.Topic)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broker publication event")
	}
}

func TestBrokerRetainedMessage(t *testing.T) {
	broker := startBroker(t)
	if err := broker.Publish("retained/value", []byte("ready"), true); err != nil {
		t.Fatalf("publish retained: %v", err)
	}

	subscriber := connectClient(t, broker.Addr(), "retained-subscriber")
	defer subscriber.Close()
	subscribe(t, subscriber, 2, "retained/#")

	pkt := readPacketWithDeadline(t, subscriber)
	if pkt.header&0x01 == 0 {
		t.Fatal("retained publication is missing the retain flag")
	}
	offset := 0
	topic, err := readMQTTString(pkt.payload, &offset)
	if err != nil {
		t.Fatalf("read retained topic: %v", err)
	}
	if topic != "retained/value" || string(pkt.payload[offset:]) != "ready" {
		t.Fatalf("retained message = %q %q", topic, pkt.payload[offset:])
	}
}

func TestBrokerPingAndCounts(t *testing.T) {
	broker := startBroker(t)
	conn := connectClient(t, broker.Addr(), "counted")
	defer conn.Close()

	if err := writePacket(conn, packetPingReq<<4, nil); err != nil {
		t.Fatalf("write ping: %v", err)
	}
	pkt := readPacketWithDeadline(t, conn)
	if pkt.header>>4 != packetPingResp {
		t.Fatalf("packet type = %d, want PINGRESP", pkt.header>>4)
	}
	if got := broker.ClientCount(); got != 1 {
		t.Fatalf("client count = %d, want 1", got)
	}
}

func TestTopicMatches(t *testing.T) {
	tests := []struct {
		filter string
		topic  string
		want   bool
	}{
		{filter: "a/b", topic: "a/b", want: true},
		{filter: "a/+/c", topic: "a/b/c", want: true},
		{filter: "a/#", topic: "a/b/c", want: true},
		{filter: "a/+", topic: "a/b/c", want: false},
		{filter: "a/b", topic: "a/c", want: false},
	}
	for _, test := range tests {
		if got := topicMatches(test.filter, test.topic); got != test.want {
			t.Errorf("topicMatches(%q, %q) = %v, want %v", test.filter, test.topic, got, test.want)
		}
	}
}

func startBroker(t *testing.T) *Broker {
	t.Helper()
	broker := New(Config{})
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

func connectClient(t *testing.T, addr, clientID string) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatalf("dial broker: %v", err)
	}

	payload := []byte{0, 4, 'M', 'Q', 'T', 'T', 4, 2, 0, 30}
	payload = appendMQTTString(payload, clientID)
	if err := writePacket(conn, packetConnect<<4, payload); err != nil {
		conn.Close()
		t.Fatalf("write CONNECT: %v", err)
	}
	pkt := readPacketWithDeadline(t, conn)
	if pkt.header>>4 != packetConnAck || len(pkt.payload) != 2 || pkt.payload[1] != 0 {
		conn.Close()
		t.Fatalf("CONNACK = %x %x", pkt.header, pkt.payload)
	}
	return conn
}

func subscribe(t *testing.T, conn net.Conn, packetID uint16, filter string) {
	t.Helper()
	payload := []byte{byte(packetID >> 8), byte(packetID)}
	payload = appendMQTTString(payload, filter)
	payload = append(payload, 0)
	if err := writePacket(conn, packetSubscribe<<4|0x02, payload); err != nil {
		t.Fatalf("write SUBSCRIBE: %v", err)
	}
	pkt := readPacketWithDeadline(t, conn)
	if pkt.header>>4 != packetSubAck {
		t.Fatalf("packet type = %d, want SUBACK", pkt.header>>4)
	}
	if len(pkt.payload) != 3 || pkt.payload[2] != 0 {
		t.Fatalf("SUBACK = %x", pkt.payload)
	}
}

func publish(t *testing.T, conn net.Conn, topic string, payload []byte) {
	t.Helper()
	body := appendMQTTString(nil, topic)
	body = append(body, payload...)
	if err := writePacket(conn, packetPublish<<4, body); err != nil {
		t.Fatalf("write PUBLISH: %v", err)
	}
}

func readPacketWithDeadline(t *testing.T, conn net.Conn) packet {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	pkt, err := readPacket(conn, defaultMaxPacketSize)
	if err != nil {
		t.Fatalf("read packet: %v", err)
	}
	return pkt
}
