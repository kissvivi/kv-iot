package mqtt

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"kv-iot/datacenter/protocol"
)

// Identity maps an MQTT topic level to the product or device identity. Values
// are "topic.<index>" or "topic.<template variable>". When omitted, the point
// binding's existing ProductKey and DeviceNo are used.
type Identity struct {
	ProductKey string `json:"productKey,omitempty"`
	DeviceNo   string `json:"deviceNo,omitempty"`
}

type RawOptions struct {
	Encoding string `json:"encoding,omitempty"` // utf8, hex, base64
}

type AckConfig struct {
	TopicTemplate    string `json:"topicTemplate"`
	CorrelationField string `json:"correlationField,omitempty"`
	TimeoutMs        int    `json:"timeoutMs,omitempty"`
}

// PointConfig is stored in protocol.PointBinding.Config. Uplink bindings
// extract a value from a payload; downlink bindings render and publish an
// action payload.
type PointConfig struct {
	Direction       string         `json:"direction,omitempty"`
	TopicTemplate   string         `json:"topicTemplate"`
	QoS             *int           `json:"qos,omitempty"`
	Identity        Identity       `json:"identity,omitempty"`
	PayloadMode     string         `json:"payloadMode,omitempty"`
	ValuePath       string         `json:"valuePath,omitempty"`
	TimestampPath   string         `json:"timestampPath,omitempty"`
	Raw             RawOptions     `json:"raw,omitempty"`
	PayloadTemplate any            `json:"payloadTemplate,omitempty"`
	Response        *AckConfig     `json:"response,omitempty"`
	Variables       map[string]any `json:"variables,omitempty"`
	Retained        bool           `json:"retained,omitempty"`
}

func ParsePointConfig(raw json.RawMessage) (PointConfig, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return PointConfig{}, fmt.Errorf("%w: point config is required", ErrInvalidBinding)
	}
	var config PointConfig
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return PointConfig{}, fmt.Errorf("%w: decode point config: %v", ErrInvalidBinding, err)
	}
	config.normalize()
	if err := config.Validate(); err != nil {
		return PointConfig{}, err
	}
	return config, nil
}

func (c *PointConfig) normalize() {
	c.Direction = strings.ToLower(strings.TrimSpace(c.Direction))
	c.TopicTemplate = strings.TrimSpace(c.TopicTemplate)
	c.PayloadMode = strings.ToLower(strings.TrimSpace(c.PayloadMode))
	c.Raw.Encoding = strings.ToLower(strings.TrimSpace(c.Raw.Encoding))
	if c.PayloadMode == "" {
		c.PayloadMode = defaultPayloadModeJSON
	}
	if c.Raw.Encoding == "" {
		c.Raw.Encoding = defaultRawEncoding
	}
	if c.Response != nil {
		c.Response.TopicTemplate = strings.TrimSpace(c.Response.TopicTemplate)
		c.Response.CorrelationField = strings.TrimSpace(c.Response.CorrelationField)
		if c.Response.CorrelationField == "" {
			c.Response.CorrelationField = "requestId"
		}
		if c.Response.TimeoutMs == 0 {
			c.Response.TimeoutMs = int(defaultAckTimeout / time.Millisecond)
		}
	}
}

func (c PointConfig) Validate() error {
	if c.Direction != "" && c.Direction != "uplink" && c.Direction != "downlink" {
		return fmt.Errorf("%w: direction must be uplink or downlink", ErrInvalidBinding)
	}
	if c.TopicTemplate == "" {
		return fmt.Errorf("%w: topicTemplate is required", ErrInvalidBinding)
	}
	if err := validateQoS(c.valueQoS(0)); err != nil {
		return err
	}
	switch c.PayloadMode {
	case "json", "json_path", "raw", "auto":
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedCodec, c.PayloadMode)
	}
	switch c.Raw.Encoding {
	case "utf8", "hex", "base64":
	default:
		return fmt.Errorf("%w: raw encoding %s", ErrUnsupportedCodec, c.Raw.Encoding)
	}
	if c.Direction != "downlink" {
		if _, err := topicTemplateToFilter(c.TopicTemplate); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidBinding, err)
		}
	}
	if c.Response != nil {
		if c.Response.TopicTemplate == "" {
			return fmt.Errorf("%w: response.topicTemplate is required", ErrInvalidBinding)
		}
		if c.Response.TimeoutMs <= 0 {
			return fmt.Errorf("%w: response.timeoutMs must be positive", ErrInvalidBinding)
		}
		if _, err := topicTemplateToFilter(c.Response.TopicTemplate); err != nil {
			return fmt.Errorf("%w: response topic: %v", ErrInvalidBinding, err)
		}
	}
	return nil
}

func (c PointConfig) valueQoS(fallback int) int {
	if c.QoS == nil {
		return fallback
	}
	return *c.QoS
}

type inboundMessage struct {
	Topic      string
	Payload    []byte
	QoS        byte
	Retained   bool
	ReceivedAt time.Time
}

func decodeUplink(point protocol.PointBinding, config PointConfig, message inboundMessage) (protocol.PointSample, error) {
	topicVariables, matched := matchTopicTemplate(config.TopicTemplate, message.Topic)
	if !matched {
		return protocol.PointSample{}, fmt.Errorf("%w: topic %q does not match %q",
			ErrInvalidTopic, message.Topic, config.TopicTemplate)
	}

	productKey, err := resolveIdentity(config.Identity.ProductKey, message.Topic, topicVariables)
	if err != nil {
		return protocol.PointSample{}, err
	}
	deviceNo, err := resolveIdentity(config.Identity.DeviceNo, message.Topic, topicVariables)
	if err != nil {
		return protocol.PointSample{}, err
	}
	if productKey == "" {
		productKey = point.ProductKey
	}
	if deviceNo == "" {
		deviceNo = point.DeviceNo
	}
	if productKey != "" && point.ProductKey != "" && productKey != point.ProductKey {
		return protocol.PointSample{}, fmt.Errorf("%w: product identity %q does not match point %q",
			ErrInvalidBinding, productKey, point.ProductKey)
	}
	if deviceNo != "" && point.DeviceNo != "" && deviceNo != point.DeviceNo {
		return protocol.PointSample{}, fmt.Errorf("%w: device identity %q does not match point %q",
			ErrInvalidBinding, deviceNo, point.DeviceNo)
	}

	value, timestamp, err := decodePayload(config, message.Payload, message.ReceivedAt)
	if err != nil {
		return protocol.PointSample{}, err
	}
	value, err = coerceValue(value, point.DataType)
	if err != nil {
		return protocol.PointSample{}, err
	}

	return protocol.PointSample{
		ProductID:       point.ProductID,
		ProductKey:      productKey,
		DeviceID:        point.DeviceID,
		DeviceNo:        deviceNo,
		PointID:         point.PointID,
		PointIdentifier: point.PointIdentifier,
		PointKind:       point.PointKind,
		DataType:        point.DataType,
		Value:           value,
		Quality:         protocol.QualityGood,
		Timestamp:       timestamp,
	}, nil
}

func decodePayload(config PointConfig, payload []byte, receivedAt time.Time) (any, time.Time, error) {
	timestamp := receivedAt
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	mode := config.PayloadMode
	if mode == "auto" {
		if json.Valid(payload) {
			mode = "json_path"
		} else {
			mode = "raw"
		}
	}
	if mode == "raw" {
		return decodeRawPayload(config.Raw.Encoding, payload), timestamp, nil
	}

	var document any
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return nil, timestamp, fmt.Errorf("%w: decode json payload: %v", ErrUnsupportedCodec, err)
	}

	value := document
	if config.ValuePath != "" {
		found, exists := getPath(document, config.ValuePath)
		if !exists {
			return nil, timestamp, fmt.Errorf("%w: valuePath %q not found",
				ErrUnsupportedCodec, config.ValuePath)
		}
		value = found
	}
	if config.TimestampPath != "" {
		found, exists := getPath(document, config.TimestampPath)
		if exists {
			if parsed, ok := parseTimestamp(found); ok {
				timestamp = parsed
			}
		}
	}
	return value, timestamp, nil
}

func decodeRawPayload(encoding string, payload []byte) any {
	switch strings.ToLower(encoding) {
	case "hex":
		return hex.EncodeToString(payload)
	case "base64":
		return base64.StdEncoding.EncodeToString(payload)
	default:
		return string(payload)
	}
}

func resolveIdentity(spec, topic string, topicVariables map[string]string) (string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", nil
	}
	if !strings.HasPrefix(spec, "topic.") {
		return spec, nil
	}
	key := strings.TrimPrefix(spec, "topic.")
	if index, err := strconv.Atoi(key); err == nil {
		levels := strings.Split(topic, "/")
		if index < 0 || index >= len(levels) {
			return "", fmt.Errorf("%w: topic index %d is out of range", ErrInvalidTopic, index)
		}
		return levels[index], nil
	}
	value, exists := topicVariables[key]
	if !exists {
		return "", fmt.Errorf("%w: topic variable %q", ErrTemplateVariable, key)
	}
	return value, nil
}

func getPath(value any, path string) (any, bool) {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, ".")
	if path == "" {
		return value, true
	}
	current := value
	for _, part := range strings.Split(path, ".") {
		switch typed := current.(type) {
		case map[string]any:
			next, exists := typed[part]
			if !exists {
				return nil, false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false
			}
			current = typed[index]
		default:
			return nil, false
		}
	}
	return current, true
}

func parseTimestamp(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case time.Time:
		return typed, true
	case json.Number:
		if number, err := typed.Int64(); err == nil {
			return timestampFromUnix(number), true
		}
		if number, err := typed.Float64(); err == nil {
			return timestampFromUnixFloat(number), true
		}
	case float64:
		return timestampFromUnixFloat(typed), true
	case int64:
		return timestampFromUnix(typed), true
	case int:
		return timestampFromUnix(int64(typed)), true
	case string:
		value := strings.TrimSpace(typed)
		if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
			return parsed, true
		}
		if number, err := strconv.ParseInt(value, 10, 64); err == nil {
			return timestampFromUnix(number), true
		}
	}
	return time.Time{}, false
}

func timestampFromUnix(value int64) time.Time {
	if value > 1_000_000_000_000 {
		return time.UnixMilli(value)
	}
	return time.Unix(value, 0)
}

func timestampFromUnixFloat(value float64) time.Time {
	if value > 1_000_000_000_000 {
		return time.UnixMilli(int64(value))
	}
	seconds := int64(value)
	nanoseconds := int64((value - float64(seconds)) * float64(time.Second))
	return time.Unix(seconds, nanoseconds)
}

func coerceValue(value any, dataType string) (any, error) {
	dataType = strings.ToLower(strings.TrimSpace(dataType))
	if dataType == "" {
		return value, nil
	}
	switch dataType {
	case "bool", "boolean":
		switch typed := value.(type) {
		case bool:
			return typed, nil
		case string:
			parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
			if err != nil {
				return nil, fmt.Errorf("%w: value %q is not bool", ErrUnsupportedCodec, typed)
			}
			return parsed, nil
		case json.Number:
			number, err := typed.Int64()
			if err != nil {
				return nil, fmt.Errorf("%w: value %q is not bool", ErrUnsupportedCodec, typed)
			}
			return number != 0, nil
		case float64:
			return typed != 0, nil
		default:
			return nil, fmt.Errorf("%w: %T is not bool", ErrUnsupportedCodec, value)
		}
	case "int", "int16", "int32", "int64", "uint", "uint16", "uint32", "uint64":
		number, err := numericInt64(value)
		if err != nil {
			return nil, err
		}
		return number, nil
	case "float", "float32", "float64", "number":
		number, err := numericFloat64(value)
		if err != nil {
			return nil, err
		}
		return number, nil
	case "string", "text":
		return valueToString(value), nil
	default:
		return value, nil
	}
}

func numericInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case json.Number:
		if number, err := typed.Int64(); err == nil {
			return number, nil
		}
		number, err := typed.Float64()
		if err != nil {
			return 0, fmt.Errorf("%w: %v", ErrUnsupportedCodec, err)
		}
		return int64(number), nil
	case float64:
		return int64(typed), nil
	case int64:
		return typed, nil
	case int:
		return int64(typed), nil
	case string:
		number, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: value %q is not integer", ErrUnsupportedCodec, typed)
		}
		return number, nil
	default:
		return 0, fmt.Errorf("%w: %T is not integer", ErrUnsupportedCodec, value)
	}
}

func numericFloat64(value any) (float64, error) {
	switch typed := value.(type) {
	case json.Number:
		number, err := typed.Float64()
		if err != nil {
			return 0, fmt.Errorf("%w: %v", ErrUnsupportedCodec, err)
		}
		return number, nil
	case float64:
		return typed, nil
	case int64:
		return float64(typed), nil
	case int:
		return float64(typed), nil
	case string:
		number, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, fmt.Errorf("%w: value %q is not numeric", ErrUnsupportedCodec, typed)
		}
		return number, nil
	default:
		return 0, fmt.Errorf("%w: %T is not numeric", ErrUnsupportedCodec, value)
	}
}
