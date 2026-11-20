package mqtt

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"kv-iot/datacenter/protocol"
)

// Message is a raw MQTT publication delivered to topic-aware stream
// subscribers.
type Message struct {
	Topic      string
	Payload    []byte
	QoS        byte
	Retained   bool
	ReceivedAt time.Time
}

type TopicHandler func(Message)

type Driver struct {
	lifecycleMu sync.Mutex
	subMu       sync.Mutex
	mu          sync.RWMutex

	driverConfig protocol.DriverConfig
	config       Config
	client       pahomqtt.Client
	connected    bool
	closed       bool
	generatedID  string

	desiredSubscriptions map[string]byte
	subscribed           map[string]byte
	inbound              []inboundMessage

	subscribers    map[uint64]chan Message
	nextSubscriber uint64
	waiters        map[string]*ackWaiter

	health protocol.DriverHealth
}

func New(config protocol.DriverConfig) (*Driver, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	parsed, err := ParseConfig(config.Config)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	driver := &Driver{
		driverConfig:         config,
		config:               parsed,
		desiredSubscriptions: make(map[string]byte),
		subscribed:           make(map[string]byte),
		subscribers:          make(map[uint64]chan Message),
		waiters:              make(map[string]*ackWaiter),
		health: protocol.DriverHealth{
			ChannelID: config.ChannelID,
			Status:    protocol.HealthUnknown,
			Since:     now,
			LastCheck: now,
		},
	}
	if parsed.ClientID == "" {
		driver.generatedID = generateClientID()
	}
	for _, subscription := range parsed.Subscriptions {
		driver.desiredSubscriptions[subscription.Topic] = byte(subscription.valueQoS(parsed.subscribeQoS()))
	}
	return driver, nil
}

// NewDriver is the registry-compatible factory.
func NewDriver(config protocol.DriverConfig) (protocol.ProtocolDriver, error) {
	return New(config)
}

func (d *Driver) Connect(ctx context.Context, config protocol.DriverConfig) error {
	d.lifecycleMu.Lock()
	defer d.lifecycleMu.Unlock()

	if err := config.Validate(); err != nil {
		return err
	}
	parsed, err := ParseConfig(config.Config)
	if err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	d.closeCurrent()
	d.mu.Lock()
	d.driverConfig = config
	d.closed = false
	d.connected = false
	d.subscribed = make(map[string]byte)
	d.health = protocol.DriverHealth{
		ChannelID: config.ChannelID,
		Status:    protocol.HealthConnecting,
		Message:   "connecting",
		Since:     time.Now(),
		LastCheck: time.Now(),
	}
	if parsed.ClientID == "" {
		if d.generatedID == "" {
			d.generatedID = generateClientID()
		}
		parsed.ClientID = d.generatedID
	}
	d.config = parsed
	for _, subscription := range parsed.Subscriptions {
		d.desiredSubscriptions[subscription.Topic] =
			byte(subscription.valueQoS(parsed.subscribeQoS()))
	}
	d.mu.Unlock()

	options := pahomqtt.NewClientOptions()
	options.AddBroker(parsed.Broker)
	options.SetClientID(parsed.ClientID)
	options.SetUsername(parsed.Username)
	options.SetPassword(parsed.Password)
	options.SetCleanSession(parsed.cleanSession())
	options.SetKeepAlive(time.Duration(parsed.KeepAliveSeconds) * time.Second)
	options.SetConnectTimeout(parsed.connectTimeout())
	options.SetConnectRetry(false)
	options.SetConnectRetryInterval(parsed.reconnectInterval())
	options.SetMaxReconnectInterval(parsed.maxReconnectInterval())
	options.SetAutoReconnect(!parsed.DisableAutoReconnect)
	options.SetOrderMatters(false)
	options.SetWriteTimeout(parsed.writeTimeout())
	options.SetProtocolVersion(uint(parsed.ProtocolVersion))
	options.SetResumeSubs(!parsed.cleanSession())
	options.SetDefaultPublishHandler(func(_ pahomqtt.Client, message pahomqtt.Message) {
		d.handleMessage(message)
	})
	options.SetOnConnectHandler(func(client pahomqtt.Client) {
		d.handleConnected(client)
	})
	options.SetConnectionLostHandler(func(client pahomqtt.Client, err error) {
		d.handleConnectionLost(client, err)
	})

	tlsConfig, err := buildTLSConfig(parsed)
	if err != nil {
		d.recordFailure(err)
		return err
	}
	if tlsConfig != nil {
		options.SetTLSConfig(tlsConfig)
	}

	client := pahomqtt.NewClient(options)
	d.mu.Lock()
	d.client = client
	d.mu.Unlock()

	token := client.Connect()
	select {
	case <-token.Done():
		if err := token.Error(); err != nil {
			d.recordFailure(err)
			d.clearClient(client)
			client.Disconnect(0)
			return fmt.Errorf("connect mqtt broker: %w", err)
		}
	case <-ctx.Done():
		client.Disconnect(0)
		d.clearClient(client)
		d.recordFailure(ctx.Err())
		return ctx.Err()
	}

	d.handleConnected(client)
	if err := d.restoreSubscriptions(client); err != nil {
		d.recordFailure(err)
		return err
	}
	return nil
}

func (d *Driver) Read(ctx context.Context, points []protocol.PointBinding) ([]protocol.PointSample, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client, err := d.connectedClient()
	if err != nil {
		return nil, err
	}
	runtimeConfig, channelID := d.configSnapshot()

	type pointDecoder struct {
		point  protocol.PointBinding
		config PointConfig
	}
	decoders := make([]pointDecoder, 0, len(points))
	for _, point := range points {
		if err := point.Validate(); err != nil {
			return nil, err
		}
		config, err := ParsePointConfig(point.Config)
		if err != nil {
			return nil, fmt.Errorf("point %s: %w", point.PointIdentifier, err)
		}
		if config.Direction == "downlink" {
			return nil, fmt.Errorf("%w: point %s is downlink-only", ErrInvalidBinding, point.PointIdentifier)
		}
		filter, err := topicTemplateToFilter(config.TopicTemplate)
		if err != nil {
			return nil, fmt.Errorf("point %s: %w", point.PointIdentifier, err)
		}
		if err := d.ensureSubscription(client, filter,
			byte(config.valueQoS(runtimeConfig.subscribeQoS()))); err != nil {
			return nil, err
		}
		decoders = append(decoders, pointDecoder{point: point, config: config})
	}

	messages := d.drainInbound()
	if len(messages) == 0 {
		return nil, nil
	}

	samples := make([]protocol.PointSample, 0, len(messages))
	for _, message := range messages {
		for _, decoder := range decoders {
			if _, matched := matchTopicTemplate(decoder.config.TopicTemplate, message.Topic); !matched {
				continue
			}
			sample, err := decodeUplink(decoder.point, decoder.config, message)
			if err != nil {
				sample = protocol.PointSample{
					ProductID:       decoder.point.ProductID,
					ProductKey:      decoder.point.ProductKey,
					DeviceID:        decoder.point.DeviceID,
					DeviceNo:        decoder.point.DeviceNo,
					PointID:         decoder.point.PointID,
					PointIdentifier: decoder.point.PointIdentifier,
					PointKind:       decoder.point.PointKind,
					DataType:        decoder.point.DataType,
					Quality:         protocol.QualityProtocolError,
					Timestamp:       message.ReceivedAt,
				}
			}
			if sample.ChannelID == "" {
				sample.ChannelID = channelID
			}
			samples = append(samples, sample)
		}
	}
	if len(samples) > 0 {
		d.recordSuccess()
	}
	return samples, nil
}

func (d *Driver) Write(ctx context.Context, point protocol.PointBinding, value any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := point.Validate(); err != nil {
		return err
	}
	client, err := d.connectedClient()
	if err != nil {
		return err
	}
	runtimeConfig, _ := d.configSnapshot()
	config, err := ParsePointConfig(point.Config)
	if err != nil {
		return fmt.Errorf("point %s: %w", point.PointIdentifier, err)
	}
	if config.Direction != "" && config.Direction != "downlink" {
		return fmt.Errorf("%w: point %s is not downlink", ErrInvalidBinding, point.PointIdentifier)
	}

	variables, requestID, params := writeVariables(point, config, value)
	topic, err := renderString(config.TopicTemplate, variables)
	if err != nil {
		return err
	}
	if err := validatePublishTopic(topic); err != nil {
		return err
	}

	payload, err := renderPublishPayload(config, params, variables)
	if err != nil {
		return err
	}
	qos := byte(config.valueQoS(runtimeConfig.publishQoS()))
	if err := validateQoS(int(qos)); err != nil {
		return err
	}

	if config.Response != nil {
		if requestID == "" {
			return fmt.Errorf("%w: requestId is required for ack correlation", ErrInvalidBinding)
		}
		ackTopic, err := renderString(config.Response.TopicTemplate, variables)
		if err != nil {
			return err
		}
		ackFilter, err := topicTemplateToFilter(config.Response.TopicTemplate)
		if err != nil {
			return err
		}
		waiter := d.registerAckWaiter(requestID, ackFilter, config.Response.CorrelationField)
		defer d.removeAckWaiter(requestID)
		if err := d.ensureSubscription(client, ackFilter,
			byte(config.valueQoS(runtimeConfig.subscribeQoS()))); err != nil {
			return err
		}
		if err := d.publish(ctx, client, topic, qos, config.Retained, payload); err != nil {
			return err
		}

		timeout := time.Duration(config.Response.TimeoutMs) * time.Millisecond
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case ackErr := <-waiter:
			if ackErr != nil {
				return ackErr
			}
			d.recordSuccess()
			return nil
		case <-timer.C:
			return fmt.Errorf("%w: topic %s request %s", ErrAckTimeout, ackTopic, requestID)
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if err := d.publish(ctx, client, topic, qos, config.Retained, payload); err != nil {
		return err
	}
	d.recordSuccess()
	return nil
}

// Subscribe implements protocol.StreamDriver. The handler receives raw payload
// bytes. Multiple subscriptions are supported and Subscribe blocks until ctx
// is canceled.
func (d *Driver) Subscribe(ctx context.Context, handler func([]byte)) error {
	if handler == nil {
		return fmt.Errorf("%w: handler is required", ErrInvalidBinding)
	}
	return d.SubscribeMessages(ctx, func(message Message) {
		handler(append([]byte(nil), message.Payload...))
	})
}

// SubscribeMessages is the topic-aware equivalent of Subscribe.
func (d *Driver) SubscribeMessages(ctx context.Context, handler TopicHandler) error {
	if handler == nil {
		return fmt.Errorf("%w: handler is required", ErrInvalidBinding)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := d.connectedClient(); err != nil {
		return err
	}

	channel, id := d.addSubscriber()
	defer d.removeSubscriber(id)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case message := <-channel:
			handler(message)
		}
	}
}

func (d *Driver) Health() protocol.DriverHealth {
	d.mu.RLock()
	health := d.health
	d.mu.RUnlock()
	health.LastCheck = time.Now()
	return health
}

func (d *Driver) Close() error {
	d.lifecycleMu.Lock()
	defer d.lifecycleMu.Unlock()
	d.closeCurrent()
	d.mu.Lock()
	d.closed = true
	d.connected = false
	d.health.Status = protocol.HealthStopped
	d.health.Message = "stopped"
	d.health.Since = time.Now()
	d.health.LastCheck = time.Now()
	d.mu.Unlock()
	return nil
}

func (d *Driver) handleConnected(client pahomqtt.Client) {
	if !d.isCurrentClient(client) {
		return
	}
	d.mu.Lock()
	d.connected = true
	d.subscribed = make(map[string]byte)
	d.health.Status = protocol.HealthHealthy
	d.health.Message = ""
	d.health.LastSuccess = time.Now()
	d.health.LastCheck = time.Now()
	d.health.ConsecutiveFailures = 0
	d.health.LastError = ""
	d.mu.Unlock()

	go func() {
		if err := d.restoreSubscriptions(client); err != nil {
			if d.isCurrentClient(client) {
				d.recordFailure(err)
			}
		}
	}()
}

func (d *Driver) handleConnectionLost(client pahomqtt.Client, err error) {
	if !d.isCurrentClient(client) {
		return
	}
	d.mu.Lock()
	d.connected = false
	d.subscribed = make(map[string]byte)
	d.mu.Unlock()
	if err == nil {
		err = fmt.Errorf("mqtt broker connection lost")
	}
	d.recordFailure(err)
}

func (d *Driver) handleMessage(message pahomqtt.Message) {
	if message == nil {
		return
	}
	d.mu.RLock()
	closed := d.closed
	d.mu.RUnlock()
	if closed {
		return
	}
	receivedAt := time.Now()
	payload := append([]byte(nil), message.Payload()...)
	event := Message{
		Topic:      message.Topic(),
		Payload:    payload,
		QoS:        message.Qos(),
		Retained:   message.Retained(),
		ReceivedAt: receivedAt,
	}
	d.enqueueInbound(inboundMessage{
		Topic:      event.Topic,
		Payload:    payload,
		QoS:        event.QoS,
		Retained:   event.Retained,
		ReceivedAt: receivedAt,
	})
	d.dispatchSubscribers(event)
	d.resolveAck(event)
	d.recordSuccess()
}

func (d *Driver) restoreSubscriptions(client pahomqtt.Client) error {
	d.subMu.Lock()
	defer d.subMu.Unlock()

	d.mu.RLock()
	desired := make(map[string]byte, len(d.desiredSubscriptions))
	for filter, qos := range d.desiredSubscriptions {
		desired[filter] = qos
	}
	d.mu.RUnlock()

	for filter, qos := range desired {
		if err := d.subscribeLocked(client, filter, qos, false); err != nil {
			return err
		}
	}
	return nil
}

func (d *Driver) ensureSubscription(client pahomqtt.Client, filter string, qos byte) error {
	if err := validateTopicFilter(filter); err != nil {
		return err
	}
	if err := validateQoS(int(qos)); err != nil {
		return err
	}
	d.subMu.Lock()
	defer d.subMu.Unlock()

	d.mu.Lock()
	d.desiredSubscriptions[filter] = qos
	currentQoS, subscribed := d.subscribed[filter]
	d.mu.Unlock()
	if subscribed && currentQoS == qos {
		return nil
	}
	return d.subscribeLocked(client, filter, qos, false)
}

func (d *Driver) subscribeLocked(client pahomqtt.Client, filter string, qos byte, force bool) error {
	d.mu.RLock()
	subscribedQoS, subscribed := d.subscribed[filter]
	d.mu.RUnlock()
	if !force && subscribed && subscribedQoS == qos {
		return nil
	}
	if !client.IsConnected() {
		return ErrNotConnected
	}
	token := client.Subscribe(filter, qos, func(_ pahomqtt.Client, message pahomqtt.Message) {
		d.handleMessage(message)
	})
	runtimeConfig, _ := d.configSnapshot()
	select {
	case <-token.Done():
		if err := token.Error(); err != nil {
			d.mu.Lock()
			delete(d.subscribed, filter)
			d.mu.Unlock()
			return fmt.Errorf("subscribe mqtt topic %s: %w", filter, err)
		}
	case <-time.After(runtimeConfig.writeTimeout()):
		d.mu.Lock()
		delete(d.subscribed, filter)
		d.mu.Unlock()
		return fmt.Errorf("%w: subscribe topic %s", ErrNotConnected, filter)
	}
	d.mu.Lock()
	d.subscribed[filter] = qos
	d.mu.Unlock()
	return nil
}

func (d *Driver) configSnapshot() (Config, string) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config, d.driverConfig.ChannelID
}

func (d *Driver) publish(ctx context.Context, client pahomqtt.Client, topic string, qos byte, retained bool, payload []byte) error {
	token := client.Publish(topic, qos, retained, payload)
	select {
	case <-token.Done():
		if err := token.Error(); err != nil {
			d.recordFailure(err)
			return fmt.Errorf("publish mqtt topic %s: %w", topic, err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *Driver) enqueueInbound(message inboundMessage) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.inbound) >= d.config.InboundQueueSize {
		copy(d.inbound, d.inbound[1:])
		d.inbound[len(d.inbound)-1] = message
		return
	}
	d.inbound = append(d.inbound, message)
}

func (d *Driver) drainInbound() []inboundMessage {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.inbound) == 0 {
		return nil
	}
	messages := make([]inboundMessage, len(d.inbound))
	copy(messages, d.inbound)
	d.inbound = d.inbound[:0]
	return messages
}

func (d *Driver) addSubscriber() (<-chan Message, uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.nextSubscriber++
	channel := make(chan Message, d.config.SubscriberQueueSize)
	d.subscribers[d.nextSubscriber] = channel
	return channel, d.nextSubscriber
}

func (d *Driver) removeSubscriber(id uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.subscribers, id)
}

func (d *Driver) dispatchSubscribers(message Message) {
	d.mu.RLock()
	channels := make([]chan Message, 0, len(d.subscribers))
	for _, channel := range d.subscribers {
		channels = append(channels, channel)
	}
	d.mu.RUnlock()
	for _, channel := range channels {
		select {
		case channel <- message:
		default:
		}
	}
}

func (d *Driver) connectedClient() (pahomqtt.Client, error) {
	d.mu.RLock()
	client := d.client
	connected := d.connected
	closed := d.closed
	d.mu.RUnlock()
	if closed {
		return nil, protocol.ErrDriverClosed
	}
	if client == nil || !connected || !client.IsConnected() {
		return nil, ErrNotConnected
	}
	return client, nil
}

func (d *Driver) isCurrentClient(client pahomqtt.Client) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.client != nil && d.client == client
}

func (d *Driver) clearClient(client pahomqtt.Client) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.client == client {
		d.client = nil
		d.connected = false
		d.subscribed = make(map[string]byte)
	}
}

func (d *Driver) closeCurrent() {
	d.mu.Lock()
	client := d.client
	d.client = nil
	d.connected = false
	d.subscribed = make(map[string]byte)
	d.mu.Unlock()
	if client != nil {
		client.Disconnect(100)
	}
}

func (d *Driver) recordSuccess() {
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return
	}
	d.health.Status = protocol.HealthHealthy
	d.health.Message = ""
	d.health.LastCheck = now
	d.health.LastSuccess = now
	d.health.ConsecutiveFailures = 0
	d.health.LastError = ""
}

func (d *Driver) recordFailure(err error) {
	if err == nil {
		return
	}
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return
	}
	d.health.Status = protocol.HealthDegraded
	d.health.Message = err.Error()
	d.health.LastCheck = now
	d.health.LastErrorAt = now
	d.health.LastError = err.Error()
	d.health.ConsecutiveFailures++
}

func buildTLSConfig(config Config) (*tls.Config, error) {
	broker := strings.ToLower(config.Broker)
	secureTransport := config.TLS.Enabled ||
		strings.HasPrefix(broker, "tls://") ||
		strings.HasPrefix(broker, "ssl://") ||
		strings.HasPrefix(broker, "wss://")
	if !secureTransport {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         config.TLS.ServerName,
		InsecureSkipVerify: config.TLS.InsecureSkipVerify, //nolint:gosec
	}
	if config.TLS.CAFile != "" {
		roots, err := x509.SystemCertPool()
		if err != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		contents, err := os.ReadFile(config.TLS.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read mqtt tls ca: %w", err)
		}
		if !roots.AppendCertsFromPEM(contents) {
			return nil, fmt.Errorf("mqtt tls ca contains no certificates")
		}
		tlsConfig.RootCAs = roots
	}
	if config.TLS.CertFile != "" {
		certificate, err := tls.LoadX509KeyPair(config.TLS.CertFile, config.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load mqtt tls client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{certificate}
	}
	return tlsConfig, nil
}

func generateClientID() string {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err == nil {
		return fmt.Sprintf("kv-iot-%x", random)
	}
	return fmt.Sprintf("kv-iot-%d", time.Now().UnixNano())
}

func renderPublishPayload(config PointConfig, params any, variables map[string]any) ([]byte, error) {
	if config.PayloadTemplate == nil {
		return marshalPublishValue(params)
	}
	rendered, err := renderTemplateValue(config.PayloadTemplate, variables)
	if err != nil {
		return nil, err
	}
	return marshalPublishValue(rendered)
}

func marshalPublishValue(value any) ([]byte, error) {
	switch typed := value.(type) {
	case []byte:
		return append([]byte(nil), typed...), nil
	case string:
		return []byte(typed), nil
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("marshal mqtt payload: %w", err)
		}
		return encoded, nil
	}
}

func writeVariables(point protocol.PointBinding, config PointConfig, value any) (map[string]any, string, any) {
	variables := map[string]any{
		"productId":        point.ProductID,
		"productKey":       point.ProductKey,
		"deviceId":         point.DeviceID,
		"deviceNo":         point.DeviceNo,
		"pointId":          point.PointID,
		"pointIdentifier":  point.PointIdentifier,
		"actionIdentifier": point.PointIdentifier,
		"timestamp":        time.Now().UTC().Format(time.RFC3339Nano),
	}
	for key, item := range config.Variables {
		variables[key] = item
	}

	params := value
	requestID := ""
	if object, ok := value.(map[string]any); ok {
		if item, exists := object["requestId"]; exists {
			requestID = valueToString(item)
		} else if item, exists := object["request_id"]; exists {
			requestID = valueToString(item)
		}
		if item, exists := object["actionIdentifier"]; exists {
			variables["actionIdentifier"] = item
		}
		if item, exists := object["params"]; exists {
			params = item
		}
		if item, exists := object["timestamp"]; exists {
			variables["timestamp"] = item
		}
		if items, exists := object["variables"].(map[string]any); exists {
			for key, item := range items {
				variables[key] = item
			}
		}
		for key, item := range object {
			if key == "variables" || key == "params" {
				continue
			}
			if _, exists := variables[key]; !exists {
				variables[key] = item
			}
		}
	}
	if config.Response != nil && requestID == "" {
		requestID = generateRequestID()
	}
	variables["requestId"] = requestID
	variables["params"] = params
	return variables, requestID, params
}

func generateRequestID() string {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err == nil {
		return fmt.Sprintf("req-%x", random)
	}
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

func decodeJSONValue(payload []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}
