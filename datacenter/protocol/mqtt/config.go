package mqtt

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"kv-iot/datacenter/protocol"
)

const (
	defaultKeepAliveSeconds     = 30
	defaultConnectTimeout       = 5 * time.Second
	defaultReconnectInterval    = 5 * time.Second
	defaultMaxReconnectInterval = 30 * time.Second
	defaultWriteTimeout         = 3 * time.Second
	defaultSubscribeQoS         = 1
	defaultPublishQoS           = 1
	defaultInboundQueueSize     = 1024
	defaultSubscriberQueueSize  = 64
	defaultAckTimeout           = 3 * time.Second
	defaultProtocolVersion      = 4
	defaultPayloadModeJSON      = "json_path"
	defaultRawEncoding          = "utf8"
)

// Subscription describes one MQTT subscription restored after every
// connection. Topic is an MQTT filter, not a concrete publish topic.
type Subscription struct {
	Topic string `json:"topic"`
	QoS   *int   `json:"qos,omitempty"`
}

// TLSConfig controls the optional TLS transport. File paths are read when the
// driver connects, not while a product configuration is being saved.
type TLSConfig struct {
	Enabled            bool   `json:"enabled"`
	CAFile             string `json:"caFile,omitempty"`
	CertFile           string `json:"certFile,omitempty"`
	KeyFile            string `json:"keyFile,omitempty"`
	ServerName         string `json:"serverName,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`
}

// Config is the connection-level MQTT driver configuration. Point-specific
// topics and payload mappings belong in PointConfig.
type Config struct {
	Broker                 string         `json:"broker"`
	ClientID               string         `json:"clientId,omitempty"`
	Username               string         `json:"username,omitempty"`
	Password               string         `json:"password,omitempty"`
	CleanSession           *bool          `json:"cleanSession,omitempty"`
	KeepAliveSeconds       int            `json:"keepAliveSeconds,omitempty"`
	ConnectTimeoutMs       int            `json:"connectTimeoutMs,omitempty"`
	ReconnectIntervalMs    int            `json:"reconnectIntervalMs,omitempty"`
	MaxReconnectIntervalMs int            `json:"maxReconnectIntervalMs,omitempty"`
	WriteTimeoutMs         int            `json:"writeTimeoutMs,omitempty"`
	ProtocolVersion        int            `json:"protocolVersion,omitempty"`
	DefaultSubscribeQoS    *int           `json:"defaultSubscribeQoS,omitempty"`
	DefaultPublishQoS      *int           `json:"defaultPublishQoS,omitempty"`
	InboundQueueSize       int            `json:"inboundQueueSize,omitempty"`
	SubscriberQueueSize    int            `json:"subscriberQueueSize,omitempty"`
	Subscriptions          []Subscription `json:"subscriptions,omitempty"`
	TLS                    TLSConfig      `json:"tls,omitempty"`
	DisableAutoReconnect   bool           `json:"disableAutoReconnect,omitempty"`
}

func ParseConfig(raw json.RawMessage) (Config, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return Config{}, fmt.Errorf("%w: config is required", protocol.ErrInvalidConfig)
	}
	var config Config
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("%w: decode mqtt config: %v", protocol.ErrInvalidConfig, err)
	}
	config.normalize()
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c *Config) normalize() {
	c.Broker = normalizeBrokerAddress(c.Broker)
	c.ClientID = strings.TrimSpace(c.ClientID)
	if c.KeepAliveSeconds == 0 {
		c.KeepAliveSeconds = defaultKeepAliveSeconds
	}
	if c.ConnectTimeoutMs == 0 {
		c.ConnectTimeoutMs = int(defaultConnectTimeout / time.Millisecond)
	}
	if c.ReconnectIntervalMs == 0 {
		c.ReconnectIntervalMs = int(defaultReconnectInterval / time.Millisecond)
	}
	if c.MaxReconnectIntervalMs == 0 {
		c.MaxReconnectIntervalMs = int(defaultMaxReconnectInterval / time.Millisecond)
	}
	if c.WriteTimeoutMs == 0 {
		c.WriteTimeoutMs = int(defaultWriteTimeout / time.Millisecond)
	}
	if c.ProtocolVersion == 0 {
		c.ProtocolVersion = defaultProtocolVersion
	}
	if c.DefaultSubscribeQoS == nil {
		value := defaultSubscribeQoS
		c.DefaultSubscribeQoS = &value
	}
	if c.DefaultPublishQoS == nil {
		value := defaultPublishQoS
		c.DefaultPublishQoS = &value
	}
	if c.InboundQueueSize == 0 {
		c.InboundQueueSize = defaultInboundQueueSize
	}
	if c.SubscriberQueueSize == 0 {
		c.SubscriberQueueSize = defaultSubscriberQueueSize
	}
	for index := range c.Subscriptions {
		c.Subscriptions[index].Topic = strings.TrimSpace(c.Subscriptions[index].Topic)
		if c.Subscriptions[index].QoS == nil {
			value := c.subscribeQoS()
			c.Subscriptions[index].QoS = &value
		}
	}
}

func (c Config) Validate() error {
	broker, err := validateBroker(c.Broker)
	if err != nil {
		return err
	}
	_ = broker
	if c.KeepAliveSeconds < 0 {
		return fmt.Errorf("%w: keepAliveSeconds must not be negative", protocol.ErrInvalidConfig)
	}
	if c.ConnectTimeoutMs < 0 ||
		c.ReconnectIntervalMs < 0 ||
		c.MaxReconnectIntervalMs < 0 ||
		c.WriteTimeoutMs < 0 {
		return fmt.Errorf("%w: timeouts must not be negative", protocol.ErrInvalidConfig)
	}
	if c.ReconnectIntervalMs > 0 &&
		c.MaxReconnectIntervalMs > 0 &&
		c.MaxReconnectIntervalMs < c.ReconnectIntervalMs {
		return fmt.Errorf("%w: maxReconnectIntervalMs must be >= reconnectIntervalMs", protocol.ErrInvalidConfig)
	}
	if c.ProtocolVersion != 3 && c.ProtocolVersion != 4 {
		return fmt.Errorf("%w: protocolVersion must be 3 or 4", protocol.ErrInvalidConfig)
	}
	if err := validateQoS(c.subscribeQoS()); err != nil {
		return err
	}
	if err := validateQoS(c.publishQoS()); err != nil {
		return err
	}
	if c.InboundQueueSize <= 0 {
		return fmt.Errorf("%w: inboundQueueSize must be positive", protocol.ErrInvalidConfig)
	}
	if c.SubscriberQueueSize <= 0 {
		return fmt.Errorf("%w: subscriberQueueSize must be positive", protocol.ErrInvalidConfig)
	}

	seen := make(map[string]int, len(c.Subscriptions))
	for index, subscription := range c.Subscriptions {
		if err := validateTopicFilter(subscription.Topic); err != nil {
			return fmt.Errorf("%w: subscriptions[%d]: %v", protocol.ErrInvalidConfig, index, err)
		}
		if err := validateQoS(subscription.valueQoS(c.subscribeQoS())); err != nil {
			return fmt.Errorf("%w: subscriptions[%d]: %v", protocol.ErrInvalidConfig, index, err)
		}
		if previousQoS, exists := seen[subscription.Topic]; exists &&
			previousQoS != subscription.valueQoS(c.subscribeQoS()) {
			return fmt.Errorf("%w: subscriptions[%d] duplicates %q with a different qos",
				protocol.ErrInvalidConfig, index, subscription.Topic)
		}
		seen[subscription.Topic] = subscription.valueQoS(c.subscribeQoS())
	}

	if c.TLS.CertFile == "" && c.TLS.KeyFile != "" ||
		c.TLS.CertFile != "" && c.TLS.KeyFile == "" {
		return fmt.Errorf("%w: tls certFile and keyFile must be configured together", protocol.ErrInvalidConfig)
	}
	return nil
}

func (c Config) cleanSession() bool {
	if c.CleanSession == nil {
		return true
	}
	return *c.CleanSession
}

func (c Config) connectTimeout() time.Duration {
	return time.Duration(c.ConnectTimeoutMs) * time.Millisecond
}

func (c Config) reconnectInterval() time.Duration {
	return time.Duration(c.ReconnectIntervalMs) * time.Millisecond
}

func (c Config) maxReconnectInterval() time.Duration {
	return time.Duration(c.MaxReconnectIntervalMs) * time.Millisecond
}

func (c Config) writeTimeout() time.Duration {
	return time.Duration(c.WriteTimeoutMs) * time.Millisecond
}

func (c Config) subscribeQoS() int {
	if c.DefaultSubscribeQoS == nil {
		return defaultSubscribeQoS
	}
	return *c.DefaultSubscribeQoS
}

func (c Config) publishQoS() int {
	if c.DefaultPublishQoS == nil {
		return defaultPublishQoS
	}
	return *c.DefaultPublishQoS
}

func (s Subscription) valueQoS(fallback int) int {
	if s.QoS == nil {
		return fallback
	}
	return *s.QoS
}

func ValidateConfig(raw json.RawMessage) error {
	_, err := ParseConfig(raw)
	return err
}

var Validator protocol.ConfigValidator = protocol.ConfigValidatorFunc(ValidateConfig)

func validateBroker(raw string) (string, error) {
	value := normalizeBrokerAddress(raw)
	if value == "" {
		return "", fmt.Errorf("%w: broker is required", ErrInvalidBroker)
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidBroker, err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "tcp", "tls", "ssl", "ws", "wss":
	default:
		return "", fmt.Errorf("%w: unsupported scheme %q", ErrInvalidBroker, parsed.Scheme)
	}
	if strings.TrimSpace(parsed.Hostname()) == "" {
		return "", fmt.Errorf("%w: broker host is required", ErrInvalidBroker)
	}
	if port := parsed.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return "", fmt.Errorf("%w: invalid port %q", ErrInvalidBroker, port)
		}
	}
	return value, nil
}

func normalizeBrokerAddress(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "://") {
		value = "tcp://" + value
	}
	return value
}

func validateQoS(qos int) error {
	if qos < 0 || qos > 2 {
		return fmt.Errorf("%w: %d", ErrInvalidQoS, qos)
	}
	return nil
}
