package tcpframe

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"kv-iot/datacenter/protocol"
)

type requestSpec struct {
	Encoding string `json:"encoding"`
	Template string `json:"template"`
	Data     string `json:"data"`
	Hex      string `json:"hex"`
}

type wireResponse struct {
	MatchPrefix      string `json:"matchPrefix"`
	MatchPrefixSnake string `json:"match_prefix"`
	MatchPrefixHex   string `json:"matchPrefixHex"`
	MatchPrefixHexS  string `json:"match_prefix_hex"`
	Fields           []wireField
}

type wireField struct {
	Identifier string `json:"identifier"`

	Offset *int `json:"offset"`
	Type   string
	Length *int

	ByteOrder      string `json:"byteOrder"`
	ByteOrderSnake string `json:"byte_order"`
	Scale          *float64
	ValueOffset    *float64 `json:"valueOffset"`
	ValueOffsetS   *float64 `json:"value_offset"`

	Bit       *int `json:"bit"`
	BitOffset *int `json:"bitOffset"`
	BitS      *int `json:"bit_offset"`
}

type wireBinding struct {
	Framing *wireFraming `json:"framing"`

	Encoding            string `json:"encoding"`
	Request             json.RawMessage
	RequestTemplate     string `json:"requestTemplate"`
	RequestTemplateS    string `json:"request_template"`
	RequestHex          string `json:"requestHex"`
	RequestHexS         string `json:"request_hex"`
	Checksum            string
	MatchPrefix         string `json:"matchPrefix"`
	MatchPrefixS        string `json:"match_prefix"`
	MatchPrefixHex      string `json:"matchPrefixHex"`
	MatchPrefixHexS     string `json:"match_prefix_hex"`
	MatchPrefixEncoding string `json:"matchPrefixEncoding"`

	Response *wireResponse `json:"response"`
	Fields   []wireField

	ExpectResponse  *bool `json:"expectResponse"`
	ExpectResponseS *bool `json:"expect_response"`
}

type fieldMapping struct {
	identifier  string
	offset      int
	kind        string
	length      int
	byteOrder   string
	scale       float64
	valueOffset float64
	bit         *int
}

type pointMapping struct {
	framing         Framing
	checksum        string
	requestTemplate string
	requestEncoding string
	requestBytes    []byte
	valueDataType   string
	matchPrefix     []byte
	fields          []fieldMapping
	expectResponse  bool
}

type FieldValue struct {
	Identifier string
	DataType   string
	Value      any
}

func parsePointBinding(point protocol.PointBinding, defaultFraming Framing, maxFrameSize int) (pointMapping, error) {
	if err := point.Validate(); err != nil {
		return pointMapping{}, err
	}
	if len(bytes.TrimSpace(point.Config)) == 0 {
		return pointMapping{}, fmt.Errorf("%w: point %s config is required", ErrInvalidBinding, point.PointIdentifier)
	}

	var wire wireBinding
	if err := json.Unmarshal(point.Config, &wire); err != nil {
		return pointMapping{}, fmt.Errorf("%w: point %s: %v", ErrInvalidBinding, point.PointIdentifier, err)
	}

	framing := defaultFraming
	if wire.Framing != nil {
		parsed, err := normalizeWireFraming(*wire.Framing)
		if err != nil {
			return pointMapping{}, fmt.Errorf("point %s framing: %w", point.PointIdentifier, err)
		}
		if err := parsed.Validate(maxFrameSize); err != nil {
			return pointMapping{}, fmt.Errorf("point %s framing: %w", point.PointIdentifier, err)
		}
		framing = parsed
	}

	checksum, err := normalizeChecksum(wire.Checksum)
	if err != nil {
		return pointMapping{}, fmt.Errorf("point %s: %w", point.PointIdentifier, err)
	}

	mapping := pointMapping{
		framing:       framing,
		checksum:      checksum,
		valueDataType: normalizeFieldType(point.DataType),
	}

	request, err := parseRequest(wire, point.PointIdentifier)
	if err != nil {
		return pointMapping{}, err
	}
	mapping.requestTemplate = request.template
	mapping.requestEncoding = request.encoding
	mapping.requestBytes = request.data

	prefix, err := parseMatchPrefix(wire)
	if err != nil {
		return pointMapping{}, fmt.Errorf("point %s: %w", point.PointIdentifier, err)
	}
	mapping.matchPrefix = prefix

	fieldConfigs := wire.Fields
	responsePresent := wire.Response != nil
	if wire.Response != nil {
		if len(fieldConfigs) == 0 {
			fieldConfigs = wire.Response.Fields
		}
		responsePrefix, err := parseMatchPrefix(wireResponseBinding(wire, *wire.Response))
		if err != nil {
			return pointMapping{}, fmt.Errorf("point %s response prefix: %w", point.PointIdentifier, err)
		}
		if len(mapping.matchPrefix) == 0 {
			mapping.matchPrefix = responsePrefix
		}
	}
	for _, field := range fieldConfigs {
		parsed, err := parseField(field)
		if err != nil {
			return pointMapping{}, fmt.Errorf("point %s: %w", point.PointIdentifier, err)
		}
		mapping.fields = append(mapping.fields, parsed)
	}

	mapping.expectResponse = len(mapping.fields) > 0 || len(mapping.matchPrefix) > 0 || responsePresent
	if wire.ExpectResponse != nil {
		mapping.expectResponse = *wire.ExpectResponse
	}
	if wire.ExpectResponseS != nil {
		if wire.ExpectResponse != nil && *wire.ExpectResponse != *wire.ExpectResponseS {
			return pointMapping{}, fmt.Errorf("%w: point %s has conflicting expectResponse fields", ErrInvalidBinding, point.PointIdentifier)
		}
		mapping.expectResponse = *wire.ExpectResponseS
	}
	return mapping, nil
}

type parsedRequest struct {
	encoding string
	template string
	data     []byte
}

func parseRequest(wire wireBinding, pointIdentifier string) (parsedRequest, error) {
	encoding := normalizeRequestEncoding(wire.Encoding)
	if encoding == "" {
		return parsedRequest{}, fmt.Errorf("%w: point %s unsupported encoding %q", ErrInvalidBinding, pointIdentifier, wire.Encoding)
	}

	if value := strings.TrimSpace(firstNonEmpty(wire.RequestHex, wire.RequestHexS)); value != "" {
		data, err := decodeHex(value)
		if err != nil {
			return parsedRequest{}, fmt.Errorf("%w: point %s requestHex: %v", ErrInvalidBinding, pointIdentifier, err)
		}
		return parsedRequest{encoding: "hex", data: data}, nil
	}
	if value := firstNonEmpty(wire.RequestTemplate, wire.RequestTemplateS); strings.TrimSpace(value) != "" {
		return parsedRequest{encoding: encoding, template: value}, nil
	}

	if len(bytes.TrimSpace(wire.Request)) > 0 {
		var text string
		if err := json.Unmarshal(wire.Request, &text); err == nil {
			return parsedRequest{encoding: encoding, template: text}, nil
		}
		var request requestSpec
		if err := json.Unmarshal(wire.Request, &request); err != nil {
			return parsedRequest{}, fmt.Errorf("%w: point %s request: %v", ErrInvalidBinding, pointIdentifier, err)
		}
		requestEncoding := normalizeRequestEncoding(firstNonEmpty(request.Encoding, wire.Encoding))
		if requestEncoding == "" {
			return parsedRequest{}, fmt.Errorf("%w: point %s unsupported request encoding %q", ErrInvalidBinding, pointIdentifier, request.Encoding)
		}
		if value := strings.TrimSpace(request.Hex); value != "" {
			data, err := decodeHex(value)
			if err != nil {
				return parsedRequest{}, fmt.Errorf("%w: point %s request hex: %v", ErrInvalidBinding, pointIdentifier, err)
			}
			return parsedRequest{encoding: "hex", data: data}, nil
		}
		if value := strings.TrimSpace(firstNonEmpty(request.Template, request.Data)); value != "" {
			if requestEncoding == "hex" {
				if containsPlaceholder(value) {
					return parsedRequest{encoding: requestEncoding, template: value}, nil
				}
				data, err := decodeHex(value)
				if err != nil {
					return parsedRequest{}, fmt.Errorf("%w: point %s request data: %v", ErrInvalidBinding, pointIdentifier, err)
				}
				return parsedRequest{encoding: "hex", data: data}, nil
			}
			return parsedRequest{encoding: requestEncoding, template: value}, nil
		}
	}
	return parsedRequest{encoding: encoding}, nil
}

func parseMatchPrefix(wire wireBinding) ([]byte, error) {
	value := firstNonEmpty(wire.MatchPrefixHex, wire.MatchPrefixHexS, wire.MatchPrefix, wire.MatchPrefixS)
	if value == "" {
		return nil, nil
	}
	if strings.TrimSpace(wire.MatchPrefixHex) != "" || strings.TrimSpace(wire.MatchPrefixHexS) != "" ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "hex:") {
		value = strings.TrimPrefix(value, "hex:")
		decoded, err := decodeHex(value)
		if err != nil {
			return nil, fmt.Errorf("matchPrefix: %w", err)
		}
		return decoded, nil
	}
	switch strings.ToLower(strings.TrimSpace(wire.MatchPrefixEncoding)) {
	case "utf8", "text", "string":
		return []byte(value), nil
	case "hex", "hexadecimal", "":
		if decoded, err := decodeHex(value); err == nil {
			return decoded, nil
		}
		return []byte(value), nil
	default:
		return nil, fmt.Errorf("unsupported matchPrefixEncoding %q", wire.MatchPrefixEncoding)
	}
}

func wireResponseBinding(wire wireBinding, response wireResponse) wireBinding {
	wire.MatchPrefix = firstNonEmpty(response.MatchPrefixHex, response.MatchPrefixHexS, response.MatchPrefix, response.MatchPrefixSnake, wire.MatchPrefix)
	wire.MatchPrefixS = ""
	wire.MatchPrefixHex = ""
	wire.MatchPrefixHexS = ""
	return wire
}

func parseField(wire wireField) (fieldMapping, error) {
	field := fieldMapping{
		identifier: strings.TrimSpace(wire.Identifier),
		kind:       normalizeFieldType(wire.Type),
		byteOrder:  normalizeFieldByteOrder(wire.ByteOrder),
		scale:      1,
	}
	if field.identifier == "" {
		return fieldMapping{}, fmt.Errorf("%w: field identifier is required", ErrInvalidBinding)
	}
	if field.kind == "" {
		return fieldMapping{}, fmt.Errorf("%w: field %s unsupported type %q", ErrInvalidBinding, field.identifier, wire.Type)
	}
	if wire.Offset != nil {
		field.offset = *wire.Offset
	}
	if field.offset < 0 {
		return fieldMapping{}, fmt.Errorf("%w: field %s offset must not be negative", ErrInvalidBinding, field.identifier)
	}
	if wire.Length != nil {
		field.length = *wire.Length
	}
	if field.length < 0 {
		return fieldMapping{}, fmt.Errorf("%w: field %s length must not be negative", ErrInvalidBinding, field.identifier)
	}
	if wire.ByteOrder != "" || wire.ByteOrderSnake != "" {
		field.byteOrder = normalizeFieldByteOrder(firstNonEmpty(wire.ByteOrder, wire.ByteOrderSnake))
		if field.byteOrder == "" {
			return fieldMapping{}, fmt.Errorf("%w: field %s byteOrder must be big or little", ErrInvalidBinding, field.identifier)
		}
	}
	if field.byteOrder == "" {
		field.byteOrder = "big"
	}
	if wire.Scale != nil {
		if *wire.Scale == 0 || math.IsNaN(*wire.Scale) || math.IsInf(*wire.Scale, 0) {
			return fieldMapping{}, fmt.Errorf("%w: field %s scale must be finite and non-zero", ErrInvalidBinding, field.identifier)
		}
		field.scale = *wire.Scale
	}
	if wire.ValueOffset != nil {
		field.valueOffset = *wire.ValueOffset
	}
	if wire.ValueOffsetS != nil {
		if wire.ValueOffset != nil && *wire.ValueOffset != *wire.ValueOffsetS {
			return fieldMapping{}, fmt.Errorf("%w: field %s has conflicting valueOffset fields", ErrInvalidBinding, field.identifier)
		}
		field.valueOffset = *wire.ValueOffsetS
	}
	if math.IsNaN(field.valueOffset) || math.IsInf(field.valueOffset, 0) {
		return fieldMapping{}, fmt.Errorf("%w: field %s valueOffset must be finite", ErrInvalidBinding, field.identifier)
	}

	bit, bitSet, err := pickInt("bit", wire.Bit, wire.BitOffset, wire.BitS)
	if err != nil {
		return fieldMapping{}, fmt.Errorf("field %s: %w", field.identifier, err)
	}
	if bitSet {
		if bit < 0 || bit > 63 {
			return fieldMapping{}, fmt.Errorf("%w: field %s bit must be between 0 and 63", ErrInvalidBinding, field.identifier)
		}
		field.bit = &bit
	}
	if field.kind != "string" && field.kind != "bytes" && field.length == 0 {
		field.length = fieldTypeSize(field.kind)
	}
	return field, nil
}

func buildRequest(mapping pointMapping, point protocol.PointBinding, value any) ([]byte, error) {
	if mapping.requestTemplate != "" {
		return renderRequestTemplate(mapping.requestTemplate, mapping.requestEncoding, mapping.valueDataType, value)
	}
	if len(mapping.requestBytes) > 0 {
		return append([]byte(nil), mapping.requestBytes...), nil
	}
	if value == nil {
		return nil, fmt.Errorf("%w: point %s request is required", ErrInvalidBinding, point.PointIdentifier)
	}
	return valueBytes(value)
}

func renderRequestTemplate(template, encoding, defaultType string, value any) ([]byte, error) {
	result := make([]byte, 0, len(template))
	for len(template) > 0 {
		index := strings.IndexByte(template, '{')
		if index < 0 {
			literal, err := decodeTemplateLiteral(template, encoding)
			if err != nil {
				return nil, err
			}
			result = append(result, literal...)
			break
		}
		if index > 0 {
			literal, err := decodeTemplateLiteral(template[:index], encoding)
			if err != nil {
				return nil, err
			}
			result = append(result, literal...)
		}
		end := strings.IndexByte(template[index:], '}')
		if end < 0 {
			return nil, fmt.Errorf("%w: unclosed request placeholder", ErrInvalidBinding)
		}
		end += index
		spec := strings.TrimSpace(template[index+1 : end])
		encoded, err := encodePlaceholder(spec, encoding, defaultType, value)
		if err != nil {
			return nil, err
		}
		result = append(result, encoded...)
		template = template[end+1:]
	}
	return result, nil
}

func decodeTemplateLiteral(value, encoding string) ([]byte, error) {
	switch encoding {
	case "utf8":
		return []byte(value), nil
	case "hex":
		decoded, err := decodeHex(value)
		if err != nil {
			return nil, fmt.Errorf("%w: request literal %q: %v", ErrInvalidBinding, value, err)
		}
		return decoded, nil
	default:
		return nil, fmt.Errorf("%w: unsupported request encoding %q", ErrInvalidBinding, encoding)
	}
}

func encodePlaceholder(spec, encoding, defaultType string, value any) ([]byte, error) {
	parts := strings.Split(spec, ":")
	name := strings.ToLower(strings.TrimSpace(parts[0]))
	if name != "value" && name != "payload" {
		return nil, fmt.Errorf("%w: unsupported request placeholder %q", ErrInvalidBinding, spec)
	}
	kind := defaultType
	endian := "big"
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		typeText := strings.ToLower(strings.TrimSpace(parts[1]))
		kind = normalizeFieldType(typeText)
		if kind == "" {
			if strings.HasSuffix(typeText, "le") {
				endian = "little"
				kind = normalizeFieldType(strings.TrimSuffix(typeText, "le"))
			} else if strings.HasSuffix(typeText, "be") {
				endian = "big"
				kind = normalizeFieldType(strings.TrimSuffix(typeText, "be"))
			}
		}
		if kind == "" {
			return nil, fmt.Errorf("%w: unsupported placeholder type %q", ErrInvalidBinding, parts[1])
		}
	}
	if len(parts) > 2 {
		if parsed := normalizeFieldByteOrder(parts[2]); parsed != "" {
			endian = parsed
		} else {
			return nil, fmt.Errorf("%w: invalid placeholder byte order %q", ErrInvalidBinding, parts[2])
		}
	}
	if kind == "" {
		switch typed := value.(type) {
		case []byte:
			return append([]byte(nil), typed...), nil
		case string:
			if encoding == "utf8" {
				return []byte(typed), nil
			}
			if strings.HasPrefix(strings.ToLower(typed), "hex:") {
				return decodeHex(typed[4:])
			}
			return nil, fmt.Errorf("%w: binary placeholder requires a type", ErrInvalidBinding)
		default:
			if encoding == "utf8" {
				return []byte(fmt.Sprint(value)), nil
			}
			return nil, fmt.Errorf("%w: binary placeholder requires a type", ErrInvalidBinding)
		}
	}

	switch kind {
	case "bytes":
		return valueBytes(value)
	case "string":
		return []byte(fmt.Sprint(value)), nil
	case "bool":
		boolean, err := boolValue(value)
		if err != nil {
			return nil, err
		}
		if boolean {
			return []byte{1}, nil
		}
		return []byte{0}, nil
	}

	size := fieldTypeSize(kind)
	if size == 0 {
		return nil, fmt.Errorf("%w: unsupported placeholder type %q", ErrInvalidBinding, kind)
	}
	target := make([]byte, size)
	switch kind {
	case "int8", "int16", "int32", "int64":
		number, err := signedValue(value)
		if err != nil {
			return nil, err
		}
		unsigned := uint64(number)
		var encoded [8]byte
		if endian == "little" {
			binary.LittleEndian.PutUint64(encoded[:], unsigned)
			copy(target, encoded[:size])
		} else {
			binary.BigEndian.PutUint64(encoded[:], unsigned)
			copy(target, encoded[len(encoded)-size:])
		}
		return target, nil
	case "uint8", "uint16", "uint32", "uint64":
		number, err := unsignedValue(value)
		if err != nil {
			return nil, err
		}
		maximum := uint64(math.MaxUint64)
		if size < 8 {
			maximum = uint64(1)<<(uint(size)*8) - 1
		}
		if number > maximum {
			return nil, fmt.Errorf("%w: value %d overflows %s", ErrInvalidBinding, number, kind)
		}
		var encoded [8]byte
		if endian == "little" {
			binary.LittleEndian.PutUint64(encoded[:], number)
			copy(target, encoded[:size])
		} else {
			binary.BigEndian.PutUint64(encoded[:], number)
			copy(target, encoded[len(encoded)-size:])
		}
		return target, nil
	case "float32":
		number, err := floatValue(value)
		if err != nil {
			return nil, err
		}
		if endian == "little" {
			binary.LittleEndian.PutUint32(target, math.Float32bits(float32(number)))
		} else {
			binary.BigEndian.PutUint32(target, math.Float32bits(float32(number)))
		}
		return target, nil
	case "float64":
		number, err := floatValue(value)
		if err != nil {
			return nil, err
		}
		if endian == "little" {
			binary.LittleEndian.PutUint64(target, math.Float64bits(number))
		} else {
			binary.BigEndian.PutUint64(target, math.Float64bits(number))
		}
		return target, nil
	default:
		return nil, fmt.Errorf("%w: unsupported placeholder type %q", ErrInvalidBinding, kind)
	}
}

func decodeFields(payload []byte, fields []fieldMapping) ([]FieldValue, error) {
	values := make([]FieldValue, 0, len(fields))
	for _, field := range fields {
		length := field.length
		if length == 0 {
			length = len(payload) - field.offset
		}
		if length < 0 || field.offset+length > len(payload) {
			return nil, fmt.Errorf("%w: field %s range [%d:%d] exceeds frame length %d",
				ErrInvalidFrame, field.identifier, field.offset, field.offset+length, len(payload))
		}
		data := append([]byte(nil), payload[field.offset:field.offset+length]...)
		if field.byteOrder == "little" {
			for left, right := 0, len(data)-1; left < right; left, right = left+1, right-1 {
				data[left], data[right] = data[right], data[left]
			}
		}

		var value any
		if field.bit != nil {
			raw, err := unsignedBytesValue(data)
			if err != nil {
				return nil, fmt.Errorf("%w: field %s bit: %v", ErrInvalidBinding, field.identifier, err)
			}
			if *field.bit >= len(data)*8 {
				return nil, fmt.Errorf("%w: field %s bit %d exceeds %d-byte value", ErrInvalidBinding, field.identifier, *field.bit, len(data))
			}
			value = raw&(uint64(1)<<uint(*field.bit)) != 0
		} else {
			switch field.kind {
			case "bool":
				value = false
				for _, item := range data {
					if item != 0 {
						value = true
						break
					}
				}
			case "string":
				value = string(data)
			case "bytes":
				value = data
			case "int8", "int16", "int32", "int64":
				raw, err := unsignedBytesValue(data)
				if err != nil {
					return nil, fmt.Errorf("field %s: %w", field.identifier, err)
				}
				value = signExtend(raw, len(data)*8)
			case "uint8", "uint16", "uint32", "uint64":
				raw, err := unsignedBytesValue(data)
				if err != nil {
					return nil, fmt.Errorf("field %s: %w", field.identifier, err)
				}
				value = raw
			case "float32":
				if len(data) != 4 {
					return nil, fmt.Errorf("%w: field %s float32 requires 4 bytes", ErrInvalidBinding, field.identifier)
				}
				value = float64(math.Float32frombits(binary.BigEndian.Uint32(data)))
			case "float64":
				if len(data) != 8 {
					return nil, fmt.Errorf("%w: field %s float64 requires 8 bytes", ErrInvalidBinding, field.identifier)
				}
				value = math.Float64frombits(binary.BigEndian.Uint64(data))
			default:
				return nil, fmt.Errorf("%w: field %s unsupported type %q", ErrInvalidBinding, field.identifier, field.kind)
			}
			if field.scale != 1 || field.valueOffset != 0 {
				number, err := floatValue(value)
				if err != nil {
					return nil, fmt.Errorf("%w: field %s scale: %v", ErrInvalidBinding, field.identifier, err)
				}
				value = number*field.scale + field.valueOffset
			}
		}
		values = append(values, FieldValue{
			Identifier: field.identifier,
			DataType:   field.kind,
			Value:      value,
		})
	}
	return values, nil
}

func normalizeFieldType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bool", "boolean", "bit":
		return "bool"
	case "i8", "int8", "sint8", "byte":
		return "int8"
	case "u8", "uint8", "usint8":
		return "uint8"
	case "i16", "int16", "short":
		return "int16"
	case "u16", "uint16", "ushort", "word":
		return "uint16"
	case "i32", "int32", "dint":
		return "int32"
	case "u32", "uint32", "udint", "dword":
		return "uint32"
	case "i64", "int64", "lint":
		return "int64"
	case "u64", "uint64", "ulint":
		return "uint64"
	case "f32", "float", "float32", "real":
		return "float32"
	case "f64", "double", "float64", "lreal":
		return "float64"
	case "string", "utf8":
		return "string"
	case "bytes", "byte_array", "raw":
		return "bytes"
	default:
		return ""
	}
}

func normalizeFieldByteOrder(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "big", "be", "big_endian", "big-endian":
		return "big"
	case "little", "le", "little_endian", "little-endian":
		return "little"
	default:
		return ""
	}
}

func normalizeRequestEncoding(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "hex", "hexadecimal":
		return "hex"
	case "utf8", "text", "string":
		return "utf8"
	default:
		return ""
	}
}

func fieldTypeSize(kind string) int {
	switch kind {
	case "bool", "int8", "uint8":
		return 1
	case "int16", "uint16":
		return 2
	case "int32", "uint32", "float32":
		return 4
	case "int64", "uint64", "float64":
		return 8
	default:
		return 0
	}
}

func decodeHex(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(strings.TrimPrefix(value, "0x"), "0X")
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", ",", "")
	value = replacer.Replace(value)
	if len(value)%2 != 0 {
		return nil, fmt.Errorf("hex value has odd length")
	}
	return hex.DecodeString(value)
}

func containsPlaceholder(value string) bool {
	return strings.Contains(value, "{") && strings.Contains(value, "}")
}

func valueBytes(value any) ([]byte, error) {
	switch typed := value.(type) {
	case []byte:
		return append([]byte(nil), typed...), nil
	case json.RawMessage:
		var text string
		if err := json.Unmarshal(typed, &text); err == nil {
			return valueBytes(text)
		}
		return append([]byte(nil), typed...), nil
	case string:
		if strings.HasPrefix(strings.ToLower(typed), "hex:") {
			return decodeHex(typed[4:])
		}
		return []byte(typed), nil
	default:
		return nil, fmt.Errorf("%w: value %T cannot be encoded as raw bytes", ErrInvalidBinding, value)
	}
}

func boolValue(value any) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		return strconv.ParseBool(typed)
	default:
		number, err := unsignedValue(value)
		if err != nil {
			return false, err
		}
		return number != 0, nil
	}
}

func unsignedValue(value any) (uint64, error) {
	if value == nil {
		return 0, fmt.Errorf("%w: value is nil", ErrInvalidBinding)
	}
	switch typed := value.(type) {
	case uint8:
		return uint64(typed), nil
	case uint16:
		return uint64(typed), nil
	case uint32:
		return uint64(typed), nil
	case uint64:
		return typed, nil
	case uint:
		return uint64(typed), nil
	case int8:
		if typed < 0 {
			return 0, fmt.Errorf("%w: negative value %d", ErrInvalidBinding, typed)
		}
		return uint64(typed), nil
	case int16:
		if typed < 0 {
			return 0, fmt.Errorf("%w: negative value %d", ErrInvalidBinding, typed)
		}
		return uint64(typed), nil
	case int32:
		if typed < 0 {
			return 0, fmt.Errorf("%w: negative value %d", ErrInvalidBinding, typed)
		}
		return uint64(typed), nil
	case int:
		if typed < 0 {
			return 0, fmt.Errorf("%w: negative value %d", ErrInvalidBinding, typed)
		}
		return uint64(typed), nil
	case int64:
		if typed < 0 {
			return 0, fmt.Errorf("%w: negative value %d", ErrInvalidBinding, typed)
		}
		return uint64(typed), nil
	case float32:
		if typed < 0 || math.Trunc(float64(typed)) != float64(typed) {
			return 0, fmt.Errorf("%w: invalid unsigned value %v", ErrInvalidBinding, typed)
		}
		return uint64(typed), nil
	case float64:
		if typed < 0 || math.Trunc(typed) != typed {
			return 0, fmt.Errorf("%w: invalid unsigned value %v", ErrInvalidBinding, typed)
		}
		return uint64(typed), nil
	case json.Number:
		number, err := strconv.ParseUint(typed.String(), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %v", ErrInvalidBinding, err)
		}
		return number, nil
	case string:
		number, err := strconv.ParseUint(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %v", ErrInvalidBinding, err)
		}
		return number, nil
	default:
		reflected := reflect.ValueOf(value)
		if reflected.IsValid() && reflected.Kind() >= reflect.Int && reflected.Kind() <= reflect.Int64 {
			return unsignedValue(reflected.Int())
		}
		if reflected.IsValid() && reflected.Kind() >= reflect.Uint && reflected.Kind() <= reflect.Uint64 {
			return reflected.Uint(), nil
		}
		return 0, fmt.Errorf("%w: unsupported numeric value %T", ErrInvalidBinding, value)
	}
}

func signedValue(value any) (int64, error) {
	if value == nil {
		return 0, fmt.Errorf("%w: value is nil", ErrInvalidBinding)
	}
	switch typed := value.(type) {
	case int8:
		return int64(typed), nil
	case int16:
		return int64(typed), nil
	case int32:
		return int64(typed), nil
	case int64:
		return typed, nil
	case int:
		return int64(typed), nil
	case uint8:
		return int64(typed), nil
	case uint16:
		return int64(typed), nil
	case uint32:
		return int64(typed), nil
	case uint64:
		if typed > math.MaxInt64 {
			return 0, fmt.Errorf("%w: value %d overflows int64", ErrInvalidBinding, typed)
		}
		return int64(typed), nil
	case json.Number:
		return typed.Int64()
	case string:
		return strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
	default:
		number, err := floatValue(value)
		if err != nil {
			return 0, err
		}
		if math.Trunc(number) != number {
			return 0, fmt.Errorf("%w: value %v is not an integer", ErrInvalidBinding, number)
		}
		return int64(number), nil
	}
}

func floatValue(value any) (float64, error) {
	switch typed := value.(type) {
	case float32:
		return float64(typed), nil
	case float64:
		return typed, nil
	case int:
		return float64(typed), nil
	case int8:
		return float64(typed), nil
	case int16:
		return float64(typed), nil
	case int32:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case uint:
		return float64(typed), nil
	case uint8:
		return float64(typed), nil
	case uint16:
		return float64(typed), nil
	case uint32:
		return float64(typed), nil
	case uint64:
		return float64(typed), nil
	case json.Number:
		return typed.Float64()
	case string:
		return strconv.ParseFloat(strings.TrimSpace(typed), 64)
	default:
		return 0, fmt.Errorf("%w: unsupported numeric value %T", ErrInvalidBinding, value)
	}
}

func unsignedBytesValue(data []byte) (uint64, error) {
	switch len(data) {
	case 1, 2, 4, 8:
		var buffer [8]byte
		copy(buffer[8-len(data):], data)
		return binary.BigEndian.Uint64(buffer[:]), nil
	default:
		return 0, fmt.Errorf("%w: bit fields require 1, 2, 4 or 8 bytes", ErrInvalidBinding)
	}
}

func signExtend(value uint64, bits int) int64 {
	if bits >= 64 {
		return int64(value)
	}
	shift := 64 - bits
	return int64(value<<uint(shift)) >> uint(shift)
}
