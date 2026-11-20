package tcpraw

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"kv-iot/datacenter/protocol"
)

const (
	ResponseModeChunk     = "chunk"
	ResponseModeFixed     = "fixed"
	ResponseModeDelimiter = "delimiter"

	defaultConnectTimeout  = 5 * time.Second
	defaultReadTimeout     = 3 * time.Second
	defaultWriteTimeout    = 3 * time.Second
	defaultResponseTimeout = 3 * time.Second
	defaultReconnectDelay  = 500 * time.Millisecond
	defaultMaxReadBytes    = 4096
	defaultMaxResponseSize = 1 << 20
)

// Config is the normalized TCP raw connection configuration.
type Config struct {
	Host                 string
	Port                 int
	ConnectTimeout       time.Duration
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	ResponseTimeout      time.Duration
	ReconnectDelay       time.Duration
	MaxReconnectAttempts int
	MaxReadBytes         int
	MaxResponseBytes     int
	ResponseMode         string
	ResponseLength       int
	ResponseDelimiter    []byte
}

type wireDuration struct {
	set   bool
	value time.Duration
}

func (d *wireDuration) UnmarshalJSON(data []byte) error {
	d.set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		value, err := time.ParseDuration(strings.TrimSpace(text))
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", text, err)
		}
		d.value = value
		return nil
	}

	var milliseconds json.Number
	if err := json.Unmarshal(data, &milliseconds); err != nil {
		return fmt.Errorf("duration must be milliseconds or a duration string: %w", err)
	}
	value, err := strconv.ParseInt(milliseconds.String(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid millisecond duration %q: %w", milliseconds.String(), err)
	}
	d.value = time.Duration(value) * time.Millisecond
	return nil
}

type wireConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`

	ConnectTimeout      *wireDuration `json:"connectTimeout"`
	ConnectTimeoutSnake *wireDuration `json:"connect_timeout"`
	ConnectTimeoutMS    *wireDuration `json:"connectTimeoutMs"`

	ReadTimeout      *wireDuration `json:"readTimeout"`
	ReadTimeoutSnake *wireDuration `json:"read_timeout"`
	ReadTimeoutMS    *wireDuration `json:"readTimeoutMs"`

	WriteTimeout      *wireDuration `json:"writeTimeout"`
	WriteTimeoutSnake *wireDuration `json:"write_timeout"`
	WriteTimeoutMS    *wireDuration `json:"writeTimeoutMs"`

	ResponseTimeout      *wireDuration `json:"responseTimeout"`
	ResponseTimeoutSnake *wireDuration `json:"response_timeout"`
	ResponseTimeoutMS    *wireDuration `json:"responseTimeoutMs"`

	Timeout      *wireDuration `json:"timeout"`
	TimeoutMS    *wireDuration `json:"timeoutMs"`
	TimeoutSnake *wireDuration `json:"timeout_ms"`

	ReconnectDelay      *wireDuration `json:"reconnectDelay"`
	ReconnectDelaySnake *wireDuration `json:"reconnect_delay"`

	MaxReconnectAttempts      *int `json:"maxReconnectAttempts"`
	MaxReconnectAttemptsSnake *int `json:"max_reconnect_attempts"`
	MaxReadBytes              *int `json:"maxReadBytes"`
	MaxReadBytesSnake         *int `json:"max_read_bytes"`
	MaxResponseBytes          *int `json:"maxResponseBytes"`
	MaxResponseBytesSnake     *int `json:"max_response_bytes"`

	ResponseMode         string `json:"responseMode"`
	ResponseModeSnake    string `json:"response_mode"`
	ResponseLength       *int   `json:"responseLength"`
	ResponseLengthSnake  *int   `json:"response_length"`
	ResponseDelimiter    string `json:"responseDelimiter"`
	ResponseDelimiterHex string `json:"responseDelimiterHex"`
}

// ValidateConfig validates a TCP raw connection configuration.
func ValidateConfig(raw json.RawMessage) error {
	_, err := parseConfig(raw)
	return err
}

// Validate checks the normalized connection configuration.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("%w: host is required", protocol.ErrInvalidConfig)
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("%w: port must be between 1 and 65535", protocol.ErrInvalidConfig)
	}
	if c.ConnectTimeout <= 0 || c.ReadTimeout <= 0 || c.WriteTimeout <= 0 || c.ResponseTimeout <= 0 {
		return fmt.Errorf("%w: connect, read, write and response timeouts must be positive", protocol.ErrInvalidConfig)
	}
	if c.ReconnectDelay < 0 {
		return fmt.Errorf("%w: reconnectDelay must not be negative", protocol.ErrInvalidConfig)
	}
	if c.MaxReconnectAttempts < 0 || c.MaxReconnectAttempts > 10 {
		return fmt.Errorf("%w: maxReconnectAttempts must be between 0 and 10", protocol.ErrInvalidConfig)
	}
	if c.MaxReadBytes < 1 || c.MaxReadBytes > defaultMaxResponseSize {
		return fmt.Errorf("%w: maxReadBytes must be between 1 and %d", protocol.ErrInvalidConfig, defaultMaxResponseSize)
	}
	if c.MaxResponseBytes < 1 || c.MaxResponseBytes > 16<<20 {
		return fmt.Errorf("%w: maxResponseBytes must be between 1 and %d", protocol.ErrInvalidConfig, 16<<20)
	}
	switch c.ResponseMode {
	case ResponseModeChunk:
	case ResponseModeFixed:
		if c.ResponseLength < 1 || c.ResponseLength > c.MaxResponseBytes {
			return fmt.Errorf("%w: responseLength must be between 1 and maxResponseBytes", protocol.ErrInvalidConfig)
		}
	case ResponseModeDelimiter:
		if len(c.ResponseDelimiter) == 0 {
			return fmt.Errorf("%w: responseDelimiter is required in delimiter mode", protocol.ErrInvalidConfig)
		}
	default:
		return fmt.Errorf("%w: responseMode must be chunk, fixed or delimiter", protocol.ErrInvalidConfig)
	}
	return nil
}

func parseConfig(raw json.RawMessage) (Config, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return Config{}, fmt.Errorf("%w: config is required", protocol.ErrInvalidConfig)
	}

	var wire wireConfig
	if err := json.Unmarshal(raw, &wire); err != nil {
		return Config{}, fmt.Errorf("%w: decode tcp raw config: %v", protocol.ErrInvalidConfig, err)
	}

	cfg := Config{
		Host:                 strings.TrimSpace(wire.Host),
		Port:                 wire.Port,
		ConnectTimeout:       defaultConnectTimeout,
		ReadTimeout:          defaultReadTimeout,
		WriteTimeout:         defaultWriteTimeout,
		ResponseTimeout:      defaultResponseTimeout,
		ReconnectDelay:       defaultReconnectDelay,
		MaxReconnectAttempts: 1,
		MaxReadBytes:         defaultMaxReadBytes,
		MaxResponseBytes:     defaultMaxResponseSize,
		ResponseMode:         ResponseModeChunk,
	}

	if value, set, err := pickDuration("timeout", wire.Timeout, wire.TimeoutMS, wire.TimeoutSnake); err != nil {
		return Config{}, err
	} else if set {
		cfg.ConnectTimeout = value
		cfg.ReadTimeout = value
		cfg.WriteTimeout = value
		cfg.ResponseTimeout = value
	}
	if value, set, err := pickDuration("connectTimeout",
		wire.ConnectTimeout, wire.ConnectTimeoutSnake, wire.ConnectTimeoutMS); err != nil {
		return Config{}, err
	} else if set {
		cfg.ConnectTimeout = value
	}
	if value, set, err := pickDuration("readTimeout",
		wire.ReadTimeout, wire.ReadTimeoutSnake, wire.ReadTimeoutMS); err != nil {
		return Config{}, err
	} else if set {
		cfg.ReadTimeout = value
	}
	if value, set, err := pickDuration("writeTimeout",
		wire.WriteTimeout, wire.WriteTimeoutSnake, wire.WriteTimeoutMS); err != nil {
		return Config{}, err
	} else if set {
		cfg.WriteTimeout = value
	}
	if value, set, err := pickDuration("responseTimeout",
		wire.ResponseTimeout, wire.ResponseTimeoutSnake, wire.ResponseTimeoutMS); err != nil {
		return Config{}, err
	} else if set {
		cfg.ResponseTimeout = value
	}
	if value, set, err := pickDuration("reconnectDelay",
		wire.ReconnectDelay, wire.ReconnectDelaySnake); err != nil {
		return Config{}, err
	} else if set {
		cfg.ReconnectDelay = value
	}

	if value, set, err := pickInt("maxReconnectAttempts",
		wire.MaxReconnectAttempts, wire.MaxReconnectAttemptsSnake); err != nil {
		return Config{}, err
	} else if set {
		cfg.MaxReconnectAttempts = value
	}
	if value, set, err := pickInt("maxReadBytes", wire.MaxReadBytes, wire.MaxReadBytesSnake); err != nil {
		return Config{}, err
	} else if set {
		cfg.MaxReadBytes = value
	}
	if value, set, err := pickInt("maxResponseBytes", wire.MaxResponseBytes, wire.MaxResponseBytesSnake); err != nil {
		return Config{}, err
	} else if set {
		cfg.MaxResponseBytes = value
	}
	if value, set, err := pickInt("responseLength", wire.ResponseLength, wire.ResponseLengthSnake); err != nil {
		return Config{}, err
	} else if set {
		cfg.ResponseLength = value
	}

	cfg.ResponseMode = normalizeResponseMode(firstNonEmpty(wire.ResponseMode, wire.ResponseModeSnake))
	if cfg.ResponseMode == "" {
		return Config{}, fmt.Errorf("%w: responseMode must be chunk, fixed or delimiter", protocol.ErrInvalidConfig)
	}

	delimiter, err := parseDelimiter(wire.ResponseDelimiter, wire.ResponseDelimiterHex)
	if err != nil {
		return Config{}, err
	}
	cfg.ResponseDelimiter = delimiter

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func parseDelimiter(value, hexValue string) ([]byte, error) {
	if strings.TrimSpace(hexValue) != "" {
		decoded, err := hex.DecodeString(strings.TrimSpace(hexValue))
		if err != nil {
			return nil, fmt.Errorf("%w: responseDelimiterHex: %v", protocol.ErrInvalidConfig, err)
		}
		return decoded, nil
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if strings.HasPrefix(strings.ToLower(value), "hex:") {
		decoded, err := hex.DecodeString(strings.TrimSpace(value[4:]))
		if err != nil {
			return nil, fmt.Errorf("%w: responseDelimiter: %v", protocol.ErrInvalidConfig, err)
		}
		return decoded, nil
	}
	return []byte(value), nil
}

func normalizeResponseMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "chunk", "read", "single":
		return ResponseModeChunk
	case "fixed", "fixed_length", "fixed-length":
		return ResponseModeFixed
	case "delimiter", "terminator", "token":
		return ResponseModeDelimiter
	default:
		return ""
	}
}

func pickInt(name string, values ...*int) (int, bool, error) {
	var (
		result int
		found  bool
	)
	for _, value := range values {
		if value == nil {
			continue
		}
		if found && result != *value {
			return 0, false, fmt.Errorf("%w: conflicting %s fields", protocol.ErrInvalidConfig, name)
		}
		result = *value
		found = true
	}
	return result, found, nil
}

func pickDuration(name string, values ...*wireDuration) (time.Duration, bool, error) {
	var (
		result time.Duration
		found  bool
	)
	for _, value := range values {
		if value == nil || !value.set {
			continue
		}
		if found && result != value.value {
			return 0, false, fmt.Errorf("%w: conflicting %s fields", protocol.ErrInvalidConfig, name)
		}
		result = value.value
		found = true
	}
	return result, found, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
