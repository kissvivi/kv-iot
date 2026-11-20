package tcpframe

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
	FramingFixed     = "fixed"
	FramingLength    = "length"
	FramingDelimiter = "delimiter"

	defaultConnectTimeout = 5 * time.Second
	defaultReadTimeout    = 3 * time.Second
	defaultWriteTimeout   = 3 * time.Second
	defaultReconnectDelay = 500 * time.Millisecond
	defaultMaxFrameSize   = 1 << 20
)

// Framing describes inbound frame boundaries and outbound request wrapping.
type Framing struct {
	Type                 string
	Length               int
	LengthOffset         int
	LengthSize           int
	Endian               string
	LengthIncludesHeader bool
	Delimiter            []byte
}

// Config is the normalized TCP frame connection configuration.
type Config struct {
	Host                 string
	Port                 int
	ConnectTimeout       time.Duration
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	ReconnectDelay       time.Duration
	MaxReconnectAttempts int
	MaxFrameSize         int
	Framing              Framing
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

type wireFraming struct {
	Type string `json:"type"`

	Length       *int `json:"length"`
	FixedLength  *int `json:"fixedLength"`
	FixedLengthS *int `json:"fixed_length"`

	LengthOffset              *int `json:"lengthOffset"`
	LengthOffsetSnake         *int `json:"length_offset"`
	LengthSize                *int `json:"lengthSize"`
	LengthSizeSnake           *int `json:"length_size"`
	Endian                    string
	LengthIncludesHeader      *bool `json:"lengthIncludesHeader"`
	LengthIncludesHeaderSnake *bool `json:"length_includes_header"`

	Delimiter     string `json:"delimiter"`
	DelimiterHex  string `json:"delimiterHex"`
	DelimiterHexS string `json:"delimiter_hex"`
}

type wireConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`

	ConnectTimeout      *wireDuration `json:"connectTimeout"`
	ConnectTimeoutSnake *wireDuration `json:"connect_timeout"`
	ConnectTimeoutMS    *wireDuration `json:"connectTimeoutMs"`
	ReadTimeout         *wireDuration `json:"readTimeout"`
	ReadTimeoutSnake    *wireDuration `json:"read_timeout"`
	ReadTimeoutMS       *wireDuration `json:"readTimeoutMs"`
	WriteTimeout        *wireDuration `json:"writeTimeout"`
	WriteTimeoutSnake   *wireDuration `json:"write_timeout"`
	WriteTimeoutMS      *wireDuration `json:"writeTimeoutMs"`
	Timeout             *wireDuration `json:"timeout"`
	TimeoutMS           *wireDuration `json:"timeoutMs"`
	TimeoutSnake        *wireDuration `json:"timeout_ms"`
	ReconnectDelay      *wireDuration `json:"reconnectDelay"`
	ReconnectDelaySnake *wireDuration `json:"reconnect_delay"`

	MaxReconnectAttempts      *int `json:"maxReconnectAttempts"`
	MaxReconnectAttemptsSnake *int `json:"max_reconnect_attempts"`
	MaxFrameSize              *int `json:"maxFrameSize"`
	MaxFrameSizeSnake         *int `json:"max_frame_size"`

	Framing *wireFraming `json:"framing"`
}

// ValidateConfig validates a TCP frame connection configuration.
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
	if c.ConnectTimeout <= 0 || c.ReadTimeout <= 0 || c.WriteTimeout <= 0 {
		return fmt.Errorf("%w: connect, read and write timeouts must be positive", protocol.ErrInvalidConfig)
	}
	if c.ReconnectDelay < 0 {
		return fmt.Errorf("%w: reconnectDelay must not be negative", protocol.ErrInvalidConfig)
	}
	if c.MaxReconnectAttempts < 0 || c.MaxReconnectAttempts > 10 {
		return fmt.Errorf("%w: maxReconnectAttempts must be between 0 and 10", protocol.ErrInvalidConfig)
	}
	if c.MaxFrameSize < 1 || c.MaxFrameSize > 16<<20 {
		return fmt.Errorf("%w: maxFrameSize must be between 1 and %d", protocol.ErrInvalidConfig, 16<<20)
	}
	return c.Framing.Validate(c.MaxFrameSize)
}

// Validate checks one framing configuration.
func (f Framing) Validate(maxFrameSize int) error {
	switch f.Type {
	case FramingFixed:
		if f.Length < 1 || f.Length > maxFrameSize {
			return fmt.Errorf("%w: fixed framing length must be between 1 and maxFrameSize", protocol.ErrInvalidConfig)
		}
	case FramingLength:
		if f.LengthOffset < 0 {
			return fmt.Errorf("%w: lengthOffset must not be negative", protocol.ErrInvalidConfig)
		}
		if f.LengthSize != 1 && f.LengthSize != 2 && f.LengthSize != 4 {
			return fmt.Errorf("%w: lengthSize must be 1, 2 or 4", protocol.ErrInvalidConfig)
		}
		if f.Endian != "big" && f.Endian != "little" {
			return fmt.Errorf("%w: framing endian must be big or little", protocol.ErrInvalidConfig)
		}
		if f.LengthOffset+f.LengthSize > maxFrameSize {
			return fmt.Errorf("%w: length header exceeds maxFrameSize", protocol.ErrInvalidConfig)
		}
	case FramingDelimiter:
		if len(f.Delimiter) == 0 {
			return fmt.Errorf("%w: delimiter is required", protocol.ErrInvalidConfig)
		}
		if len(f.Delimiter) > maxFrameSize {
			return fmt.Errorf("%w: delimiter exceeds maxFrameSize", protocol.ErrInvalidConfig)
		}
	default:
		return fmt.Errorf("%w: framing type must be fixed, length or delimiter", protocol.ErrInvalidConfig)
	}
	return nil
}

func parseConfig(raw json.RawMessage) (Config, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return Config{}, fmt.Errorf("%w: config is required", protocol.ErrInvalidConfig)
	}
	var wire wireConfig
	if err := json.Unmarshal(raw, &wire); err != nil {
		return Config{}, fmt.Errorf("%w: decode tcp frame config: %v", protocol.ErrInvalidConfig, err)
	}

	cfg := Config{
		Host:                 strings.TrimSpace(wire.Host),
		Port:                 wire.Port,
		ConnectTimeout:       defaultConnectTimeout,
		ReadTimeout:          defaultReadTimeout,
		WriteTimeout:         defaultWriteTimeout,
		ReconnectDelay:       defaultReconnectDelay,
		MaxReconnectAttempts: 1,
		MaxFrameSize:         defaultMaxFrameSize,
	}
	if value, set, err := pickDuration("timeout", wire.Timeout, wire.TimeoutMS, wire.TimeoutSnake); err != nil {
		return Config{}, err
	} else if set {
		cfg.ConnectTimeout = value
		cfg.ReadTimeout = value
		cfg.WriteTimeout = value
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
	if value, set, err := pickInt("maxFrameSize", wire.MaxFrameSize, wire.MaxFrameSizeSnake); err != nil {
		return Config{}, err
	} else if set {
		cfg.MaxFrameSize = value
	}
	if wire.Framing == nil {
		return Config{}, fmt.Errorf("%w: framing is required", protocol.ErrInvalidConfig)
	}
	framing, err := normalizeWireFraming(*wire.Framing)
	if err != nil {
		return Config{}, err
	}
	cfg.Framing = framing
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func normalizeWireFraming(wire wireFraming) (Framing, error) {
	framing := Framing{
		Type:       normalizeFramingType(wire.Type),
		LengthSize: 2,
		Endian:     normalizeEndian(wire.Endian),
	}
	if framing.Type == "" {
		return Framing{}, fmt.Errorf("%w: unsupported framing type %q", protocol.ErrInvalidConfig, wire.Type)
	}
	if framing.Endian == "" {
		return Framing{}, fmt.Errorf("%w: framing endian must be big or little", protocol.ErrInvalidConfig)
	}

	length, set, err := pickInt("fixedLength", wire.FixedLength, wire.FixedLengthS, wire.Length)
	if err != nil {
		return Framing{}, err
	}
	if set {
		framing.Length = length
	}
	if value, set, err := pickInt("lengthOffset", wire.LengthOffset, wire.LengthOffsetSnake); err != nil {
		return Framing{}, err
	} else if set {
		framing.LengthOffset = value
	}
	if value, set, err := pickInt("lengthSize", wire.LengthSize, wire.LengthSizeSnake); err != nil {
		return Framing{}, err
	} else if set {
		framing.LengthSize = value
	}
	if wire.LengthIncludesHeader != nil {
		framing.LengthIncludesHeader = *wire.LengthIncludesHeader
	}
	if wire.LengthIncludesHeaderSnake != nil {
		value := *wire.LengthIncludesHeaderSnake
		if wire.LengthIncludesHeader != nil && *wire.LengthIncludesHeader != value {
			return Framing{}, fmt.Errorf("%w: conflicting lengthIncludesHeader fields", protocol.ErrInvalidConfig)
		}
		framing.LengthIncludesHeader = value
	}

	delimiter := firstNonEmpty(wire.DelimiterHex, wire.DelimiterHexS)
	if delimiter != "" {
		decoded, err := hex.DecodeString(strings.TrimSpace(delimiter))
		if err != nil {
			return Framing{}, fmt.Errorf("%w: delimiterHex: %v", protocol.ErrInvalidConfig, err)
		}
		framing.Delimiter = decoded
	} else if strings.TrimSpace(wire.Delimiter) != "" {
		value := strings.TrimSpace(wire.Delimiter)
		if strings.HasPrefix(strings.ToLower(value), "hex:") {
			decoded, err := hex.DecodeString(strings.TrimSpace(value[4:]))
			if err != nil {
				return Framing{}, fmt.Errorf("%w: delimiter: %v", protocol.ErrInvalidConfig, err)
			}
			framing.Delimiter = decoded
		} else {
			framing.Delimiter = []byte(value)
		}
	}
	return framing, nil
}

func normalizeFramingType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "fixed", "fixed_length", "fixed-length", "fixedlen":
		return FramingFixed
	case "length", "length_field", "length-field", "length_prefix", "length-prefix":
		return FramingLength
	case "delimiter", "terminator", "token":
		return FramingDelimiter
	default:
		return ""
	}
}

func normalizeEndian(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "big", "be", "big_endian", "big-endian":
		return "big"
	case "little", "le", "little_endian", "little-endian":
		return "little"
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
