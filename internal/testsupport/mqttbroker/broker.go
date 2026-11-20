// Package mqttbroker provides a minimal in-process MQTT 3.1.1 broker for
// protocol integration tests. It supports CONNECT, SUBSCRIBE, UNSUBSCRIBE,
// QoS 0/1 PUBLISH, retained messages, PING, and DISCONNECT.
package mqttbroker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"kv-iot/internal/testsupport/tcpserver"
)

const defaultMaxPacketSize = 1 << 20

// Config controls a test broker.
type Config struct {
	Addr          string
	MaxPacketSize int
	EventBuffer   int
}

// Message is a published MQTT message.
type Message struct {
	Topic    string
	Payload  []byte
	Retained bool
}

// Broker is a startable minimal MQTT broker.
type Broker struct {
	base          *tcpserver.Server
	maxPacketSize int
	events        chan Message

	mu       sync.RWMutex
	clients  map[*client]struct{}
	retained map[string]Message
}

type client struct {
	id   string
	conn net.Conn

	writeMu sync.Mutex
	mu      sync.RWMutex
	subs    map[string]byte
}

// New creates a broker. The default listen address is 127.0.0.1:0.
func New(cfg Config) *Broker {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:0"
	}
	if cfg.MaxPacketSize <= 0 {
		cfg.MaxPacketSize = defaultMaxPacketSize
	}
	if cfg.EventBuffer <= 0 {
		cfg.EventBuffer = 256
	}
	broker := &Broker{
		maxPacketSize: cfg.MaxPacketSize,
		events:        make(chan Message, cfg.EventBuffer),
		clients:       make(map[*client]struct{}),
		retained:      make(map[string]Message),
	}
	broker.base = tcpserver.NewWithConfig(tcpserver.Config{Addr: cfg.Addr}, broker.handleConnection)
	return broker
}

// Start begins accepting MQTT clients.
func (b *Broker) Start() error {
	return b.base.Start()
}

// Addr returns the actual listening address after Start.
func (b *Broker) Addr() string {
	return b.base.Addr()
}

// Close stops the broker and all active client connections.
func (b *Broker) Close() error {
	return b.base.Close()
}

// Published exposes recent broker publications for assertions. The channel is
// bounded and never blocks the broker; events are dropped if the buffer is full.
func (b *Broker) Published() <-chan Message {
	return b.events
}

// Publish injects a message into the broker and delivers it to matching
// subscribers.
func (b *Broker) Publish(topic string, payload []byte, retained bool) error {
	if topic == "" || strings.ContainsAny(topic, "+#") {
		return fmt.Errorf("mqttbroker: invalid publish topic %q", topic)
	}
	message := Message{
		Topic:    topic,
		Payload:  append([]byte(nil), payload...),
		Retained: retained,
	}
	b.mu.Lock()
	if retained {
		if len(payload) == 0 {
			delete(b.retained, topic)
		} else {
			b.retained[topic] = message
		}
	}
	clients := make([]*client, 0, len(b.clients))
	for c := range b.clients {
		clients = append(clients, c)
	}
	b.mu.Unlock()

	select {
	case b.events <- message:
	default:
	}
	b.deliver(clients, message, false)
	return nil
}

// ClientCount returns the number of CONNECTed clients.
func (b *Broker) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// SubscriptionCount returns the total number of active subscriptions.
func (b *Broker) SubscriptionCount() int {
	b.mu.RLock()
	clients := make([]*client, 0, len(b.clients))
	for c := range b.clients {
		clients = append(clients, c)
	}
	b.mu.RUnlock()

	total := 0
	for _, c := range clients {
		c.mu.RLock()
		total += len(c.subs)
		c.mu.RUnlock()
	}
	return total
}

func (b *Broker) handleConnection(_ context.Context, conn net.Conn) error {
	c := &client{
		conn: conn,
		subs: make(map[string]byte),
	}
	connected := false
	defer func() {
		if connected {
			b.mu.Lock()
			delete(b.clients, c)
			b.mu.Unlock()
		}
	}()

	for {
		pkt, err := readPacket(conn, b.maxPacketSize)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		packetType := pkt.header >> 4

		switch packetType {
		case packetConnect:
			if connected {
				return errors.New("mqttbroker: duplicate CONNECT")
			}
			clientID, err := parseConnect(pkt.payload)
			if err != nil {
				return err
			}
			c.id = clientID
			if err := writePacket(conn, packetConnAck<<4, []byte{0, 0}); err != nil {
				return err
			}
			b.mu.Lock()
			b.clients[c] = struct{}{}
			b.mu.Unlock()
			connected = true

		case packetPublish:
			if !connected {
				return errors.New("mqttbroker: PUBLISH before CONNECT")
			}
			if err := b.handlePublish(c, pkt); err != nil {
				return err
			}

		case packetSubscribe:
			if !connected {
				return errors.New("mqttbroker: SUBSCRIBE before CONNECT")
			}
			if err := b.handleSubscribe(c, pkt.payload); err != nil {
				return err
			}

		case packetUnsubscribe:
			if !connected {
				return errors.New("mqttbroker: UNSUBSCRIBE before CONNECT")
			}
			if err := handleUnsubscribe(c, pkt.payload); err != nil {
				return err
			}

		case packetPingReq:
			if err := c.write(packetPingResp<<4, nil); err != nil {
				return err
			}

		case packetPubAck:
			// Client acknowledgements are not needed because this broker sends QoS 0.

		case packetDisconnect:
			return nil

		default:
			return fmt.Errorf("mqttbroker: unsupported packet type %d", packetType)
		}
	}
}

func parseConnect(payload []byte) (string, error) {
	offset := 0
	protocolName, err := readMQTTString(payload, &offset)
	if err != nil {
		return "", err
	}
	if protocolName != "MQTT" && protocolName != "MQIsdp" {
		return "", fmt.Errorf("mqttbroker: unsupported protocol %q", protocolName)
	}
	if offset+4 > len(payload) {
		return "", errors.New("mqttbroker: truncated CONNECT")
	}
	protocolLevel := payload[offset]
	if protocolLevel != 3 && protocolLevel != 4 {
		return "", fmt.Errorf("mqttbroker: unsupported protocol level %d", protocolLevel)
	}
	offset += 4 // protocol level, connect flags, keep-alive.
	clientID, err := readMQTTString(payload, &offset)
	if err != nil {
		return "", err
	}
	return clientID, nil
}

func (b *Broker) handlePublish(c *client, pkt packet) error {
	qos := (pkt.header >> 1) & 0x03
	if qos > 1 {
		return fmt.Errorf("mqttbroker: unsupported publish qos %d", qos)
	}
	offset := 0
	topic, err := readMQTTString(pkt.payload, &offset)
	if err != nil {
		return err
	}
	var packetID uint16
	if qos == 1 {
		if offset+2 > len(pkt.payload) {
			return errors.New("mqttbroker: truncated publish packet id")
		}
		packetID = uint16(pkt.payload[offset])<<8 | uint16(pkt.payload[offset+1])
		offset += 2
	}
	if err := b.Publish(topic, pkt.payload[offset:], pkt.header&0x01 != 0); err != nil {
		return err
	}
	if qos == 1 {
		return c.write(packetPubAck<<4, []byte{byte(packetID >> 8), byte(packetID)})
	}
	return nil
}

func (b *Broker) handleSubscribe(c *client, payload []byte) error {
	if len(payload) < 2 {
		return errors.New("mqttbroker: truncated SUBSCRIBE")
	}
	packetID := []byte{payload[0], payload[1]}
	offset := 2
	var granted []byte
	for offset < len(payload) {
		filter, err := readMQTTString(payload, &offset)
		if err != nil {
			return err
		}
		if filter == "" || offset >= len(payload) {
			return errors.New("mqttbroker: invalid SUBSCRIBE payload")
		}
		requestedQoS := payload[offset] & 0x03
		offset++
		if requestedQoS > 2 {
			return errors.New("mqttbroker: invalid subscription qos")
		}

		c.mu.Lock()
		c.subs[filter] = 0
		c.mu.Unlock()
		granted = append(granted, 0)
	}
	if len(granted) == 0 {
		return errors.New("mqttbroker: empty SUBSCRIBE")
	}

	response := append([]byte(nil), packetID...)
	response = append(response, granted...)
	if err := c.write(packetSubAck<<4, response); err != nil {
		return err
	}

	// Retained messages are delivered after SUBACK, as required by MQTT.
	b.mu.RLock()
	retained := make([]Message, 0, len(b.retained))
	for _, message := range b.retained {
		if c.matches(message.Topic) {
			retained = append(retained, message)
		}
	}
	b.mu.RUnlock()
	for _, message := range retained {
		if err := c.publish(message, true); err != nil {
			return err
		}
	}
	return nil
}

func handleUnsubscribe(c *client, payload []byte) error {
	if len(payload) < 2 {
		return errors.New("mqttbroker: truncated UNSUBSCRIBE")
	}
	packetID := []byte{payload[0], payload[1]}
	offset := 2
	for offset < len(payload) {
		filter, err := readMQTTString(payload, &offset)
		if err != nil {
			return err
		}
		c.mu.Lock()
		delete(c.subs, filter)
		c.mu.Unlock()
	}
	return c.write(packetUnsubAck<<4, packetID)
}

func (b *Broker) deliver(clients []*client, message Message, retained bool) {
	for _, c := range clients {
		if c.matches(message.Topic) {
			_ = c.publish(message, retained)
		}
	}
}

func (c *client) publish(message Message, retained bool) error {
	payload := make([]byte, 0, 2+len(message.Topic)+len(message.Payload))
	payload = appendMQTTString(payload, message.Topic)
	payload = append(payload, message.Payload...)
	header := packetPublish << 4
	if retained {
		header |= 0x01
	}
	return c.write(header, payload)
}

func (c *client) write(header byte, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return writePacket(c.conn, header, payload)
}

func (c *client) matches(topic string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for filter := range c.subs {
		if topicMatches(filter, topic) {
			return true
		}
	}
	return false
}

func topicMatches(filter, topic string) bool {
	filterParts := strings.Split(filter, "/")
	topicParts := strings.Split(topic, "/")
	for i, filterPart := range filterParts {
		if filterPart == "#" {
			return i == len(filterParts)-1
		}
		if i >= len(topicParts) {
			return false
		}
		if filterPart == "+" {
			continue
		}
		if filterPart != topicParts[i] {
			return false
		}
	}
	return len(filterParts) == len(topicParts)
}
