package tcpraw

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kv-iot/datacenter/protocol"
)

type wirePointBinding struct {
	Encoding        string `json:"encoding"`
	Request         string `json:"request"`
	RequestHex      string `json:"requestHex"`
	RequestHexSnake string `json:"request_hex"`
}

func requestPayload(raw json.RawMessage) ([]byte, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, fmt.Errorf("%w: request is required", ErrInvalidBinding)
	}
	var binding wirePointBinding
	if err := json.Unmarshal(raw, &binding); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBinding, err)
	}

	value := firstNonEmpty(binding.RequestHex, binding.RequestHexSnake)
	if value != "" {
		decoded, err := decodeHex(value)
		if err != nil {
			return nil, fmt.Errorf("%w: requestHex: %v", ErrInvalidBinding, err)
		}
		return decoded, nil
	}

	value = strings.TrimSpace(binding.Request)
	if value == "" {
		return nil, fmt.Errorf("%w: request or requestHex is required", ErrInvalidBinding)
	}
	switch strings.ToLower(strings.TrimSpace(binding.Encoding)) {
	case "hex", "hexadecimal", "":
		decoded, err := decodeHex(value)
		if err != nil {
			return nil, fmt.Errorf("%w: request: %v", ErrInvalidBinding, err)
		}
		return decoded, nil
	case "utf8", "text", "string":
		return []byte(value), nil
	default:
		return nil, fmt.Errorf("%w: unsupported encoding %q", ErrInvalidBinding, binding.Encoding)
	}
}

func decodeHex(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(strings.TrimPrefix(value, "0x"), "0X")
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "-", "")
	if len(value)%2 != 0 {
		return nil, fmt.Errorf("hex value has odd length")
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func bytesValue(value any) ([]byte, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case []byte:
		return append([]byte(nil), typed...), nil
	case string:
		if strings.HasPrefix(strings.ToLower(typed), "hex:") {
			return decodeHex(typed[4:])
		}
		return []byte(typed), nil
	case json.RawMessage:
		var text string
		if err := json.Unmarshal(typed, &text); err == nil {
			return bytesValue(text)
		}
		return append([]byte(nil), typed...), nil
	default:
		return nil, fmt.Errorf("%w: write value must be []byte or string", ErrInvalidBinding)
	}
}

func bindingSample(point protocol.PointBinding, channelID string, timestamp time.Time, value []byte) protocol.PointSample {
	return protocol.PointSample{
		ChannelID:       channelID,
		ProductID:       point.ProductID,
		ProductKey:      point.ProductKey,
		DeviceID:        point.DeviceID,
		DeviceNo:        point.DeviceNo,
		PointID:         point.PointID,
		PointIdentifier: point.PointIdentifier,
		PointKind:       point.PointKind,
		DataType:        "bytes",
		Value:           append([]byte(nil), value...),
		Quality:         protocol.QualityGood,
		Timestamp:       timestamp,
	}
}
