package modbus

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"kv-iot/datacenter/protocol"
)

const (
	areaCoil             = "coil"
	areaDiscreteInput    = "discrete_input"
	areaHoldingRegister  = "holding_register"
	areaInputRegister    = "input_register"
	typeBool             = "bool"
	typeInt8             = "int8"
	typeUint8            = "uint8"
	typeInt16            = "int16"
	typeUint16           = "uint16"
	typeInt32            = "int32"
	typeUint32           = "uint32"
	typeInt64            = "int64"
	typeUint64           = "uint64"
	typeFloat32          = "float32"
	typeFloat64          = "float64"
	typeString           = "string"
	typeBytes            = "bytes"
	orderBig             = "big"
	orderLittle          = "little"
	fcReadCoils          = byte(0x01)
	fcReadDiscreteInputs = byte(0x02)
	fcReadHolding        = byte(0x03)
	fcReadInput          = byte(0x04)
	fcWriteCoil          = byte(0x05)
	fcWriteRegister      = byte(0x06)
)

type wireBinding struct {
	Area     string `json:"area"`
	Address  *int   `json:"address"`
	Quantity *int   `json:"quantity"`
	Length   *int   `json:"length"`

	DataType      string `json:"dataType"`
	DataTypeSnake string `json:"data_type"`

	ByteOrder      string `json:"byteOrder"`
	ByteOrderSnake string `json:"byte_order"`
	WordOrder      string `json:"wordOrder"`
	WordOrderSnake string `json:"word_order"`

	Scale  *float64 `json:"scale"`
	Offset *float64 `json:"offset"`

	BitOffset      *int `json:"bitOffset"`
	BitOffsetSnake *int `json:"bit_offset"`
	UnitID         *int `json:"unitId"`
	UnitIDSnake    *int `json:"unit_id"`
}

type pointMapping struct {
	area        string
	address     int
	quantity    int
	unitID      int
	dataType    string
	byteOrder   string
	wordOrder   string
	scale       float64
	offset      float64
	bitOffset   *int
	readFunc    byte
	writable    bool
	registerLen int
}

// ValidatePointBinding validates one point-level Modbus configuration without
// opening a connection.
func ValidatePointBinding(config json.RawMessage) error {
	_, err := parsePointBinding(protocol.PointBinding{
		PointIdentifier: "validation",
		Config:          config,
	}, defaultUnitID)
	return err
}

func parsePointBinding(point protocol.PointBinding, defaultUnitID int) (pointMapping, error) {
	if err := point.Validate(); err != nil {
		return pointMapping{}, err
	}
	if len(strings.TrimSpace(string(point.Config))) == 0 {
		return pointMapping{}, fmt.Errorf("%w: point %s config is required", ErrInvalidBinding, point.PointIdentifier)
	}

	var wire wireBinding
	if err := json.Unmarshal(point.Config, &wire); err != nil {
		return pointMapping{}, fmt.Errorf("%w: point %s: %v", ErrInvalidBinding, point.PointIdentifier, err)
	}

	area := normalizeArea(wire.Area)
	if area == "" {
		return pointMapping{}, fmt.Errorf("%w: point %s area %q", ErrInvalidBinding, point.PointIdentifier, wire.Area)
	}
	if wire.Address == nil {
		return pointMapping{}, fmt.Errorf("%w: point %s address is required", ErrInvalidBinding, point.PointIdentifier)
	}
	if *wire.Address < 0 || *wire.Address > 65535 {
		return pointMapping{}, fmt.Errorf("%w: point %s address must be between 0 and 65535", ErrInvalidBinding, point.PointIdentifier)
	}

	unitID, unitSet, err := pickInt("unitId", wire.UnitID, wire.UnitIDSnake)
	if err != nil {
		return pointMapping{}, fmt.Errorf("%w: point %s: %v", ErrInvalidBinding, point.PointIdentifier, err)
	}
	if !unitSet {
		unitID = defaultUnitID
	}
	if unitID < 0 || unitID > 255 {
		return pointMapping{}, fmt.Errorf("%w: point %s unitId must be between 0 and 255", ErrInvalidBinding, point.PointIdentifier)
	}

	rawDataType := firstNonEmpty(wire.DataType, wire.DataTypeSnake)
	dataType := normalizeDataType(rawDataType)
	if rawDataType != "" && dataType == "" {
		return pointMapping{}, fmt.Errorf("%w: point %s dataType %q", ErrUnsupportedType, point.PointIdentifier, rawDataType)
	}
	if rawDataType == "" {
		if area == areaCoil || area == areaDiscreteInput {
			dataType = typeBool
		} else {
			dataType = typeUint16
		}
	}

	rawByteOrder := firstNonEmpty(wire.ByteOrder, wire.ByteOrderSnake)
	byteOrder := normalizeOrder(rawByteOrder)
	if rawByteOrder == "" {
		byteOrder = orderBig
	}
	if byteOrder != orderBig && byteOrder != orderLittle {
		return pointMapping{}, fmt.Errorf("%w: point %s byteOrder must be big or little", ErrInvalidBinding, point.PointIdentifier)
	}
	rawWordOrder := firstNonEmpty(wire.WordOrder, wire.WordOrderSnake)
	wordOrder := normalizeOrder(rawWordOrder)
	if rawWordOrder == "" {
		wordOrder = orderBig
	}
	if wordOrder != orderBig && wordOrder != orderLittle {
		return pointMapping{}, fmt.Errorf("%w: point %s wordOrder must be big or little", ErrInvalidBinding, point.PointIdentifier)
	}

	bitOffset, bitSet, err := pickInt("bitOffset", wire.BitOffset, wire.BitOffsetSnake)
	if err != nil {
		return pointMapping{}, fmt.Errorf("%w: point %s: %v", ErrInvalidBinding, point.PointIdentifier, err)
	}
	if bitSet && (bitOffset < 0 || bitOffset > 15) {
		return pointMapping{}, fmt.Errorf("%w: point %s bitOffset must be between 0 and 15", ErrInvalidBinding, point.PointIdentifier)
	}
	if bitSet && area != areaHoldingRegister {
		return pointMapping{}, fmt.Errorf("%w: point %s bitOffset requires holding_register area", ErrInvalidBinding, point.PointIdentifier)
	}

	quantity, quantitySet, err := pickInt("quantity", wire.Quantity, wire.Length)
	if err != nil {
		return pointMapping{}, fmt.Errorf("%w: point %s: %v", ErrInvalidBinding, point.PointIdentifier, err)
	}
	if quantitySet && quantity <= 0 {
		return pointMapping{}, fmt.Errorf("%w: point %s quantity must be positive", ErrInvalidBinding, point.PointIdentifier)
	}

	requiredRegisters := dataTypeRegisterCount(dataType)
	if area == areaCoil || area == areaDiscreteInput {
		if dataType != typeBool {
			return pointMapping{}, fmt.Errorf("%w: point %s bit areas require bool dataType", ErrInvalidBinding, point.PointIdentifier)
		}
		if quantitySet && quantity != 1 {
			return pointMapping{}, fmt.Errorf("%w: point %s bit areas require quantity 1", ErrInvalidBinding, point.PointIdentifier)
		}
		quantity = 1
	} else {
		if requiredRegisters == 0 && !quantitySet {
			return pointMapping{}, fmt.Errorf("%w: point %s dataType %s requires quantity", ErrInvalidBinding, point.PointIdentifier, dataType)
		}
		if !quantitySet {
			quantity = requiredRegisters
		}
		if quantity > 125 {
			return pointMapping{}, fmt.Errorf("%w: point %s quantity must not exceed 125 registers", ErrInvalidBinding, point.PointIdentifier)
		}
		if requiredRegisters > 0 && quantity < requiredRegisters {
			return pointMapping{}, fmt.Errorf("%w: point %s quantity %d is too small for %s", ErrInvalidBinding, point.PointIdentifier, quantity, dataType)
		}
	}

	scale := 1.0
	if wire.Scale != nil {
		scale = *wire.Scale
	}
	if scale == 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return pointMapping{}, fmt.Errorf("%w: point %s scale must be finite and non-zero", ErrInvalidBinding, point.PointIdentifier)
	}
	offset := 0.0
	if wire.Offset != nil {
		offset = *wire.Offset
		if math.IsNaN(offset) || math.IsInf(offset, 0) {
			return pointMapping{}, fmt.Errorf("%w: point %s offset must be finite", ErrInvalidBinding, point.PointIdentifier)
		}
	}

	mapping := pointMapping{
		area:        area,
		address:     *wire.Address,
		quantity:    quantity,
		unitID:      unitID,
		dataType:    dataType,
		byteOrder:   byteOrder,
		wordOrder:   wordOrder,
		scale:       scale,
		offset:      offset,
		registerLen: requiredRegisters,
	}
	if bitSet {
		value := bitOffset
		mapping.bitOffset = &value
	}
	switch area {
	case areaCoil:
		mapping.readFunc = fcReadCoils
		mapping.writable = true
	case areaDiscreteInput:
		mapping.readFunc = fcReadDiscreteInputs
	case areaHoldingRegister:
		mapping.readFunc = fcReadHolding
		mapping.writable = true
	case areaInputRegister:
		mapping.readFunc = fcReadInput
	}
	return mapping, nil
}

func normalizeArea(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "coil", "coils":
		if strings.TrimSpace(value) == "" {
			return ""
		}
		return areaCoil
	case "discrete", "discrete_input", "discrete_inputs", "input_bit":
		return areaDiscreteInput
	case "holding", "holding_register", "holding_registers", "hr":
		return areaHoldingRegister
	case "input", "input_register", "input_registers", "ir":
		return areaInputRegister
	default:
		return ""
	}
}

func normalizeDataType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bool", "boolean", "bit":
		return typeBool
	case "int8", "sint8", "byte":
		return typeInt8
	case "uint8", "usint8":
		return typeUint8
	case "int16", "int", "short":
		return typeInt16
	case "uint16", "uint", "ushort", "word":
		return typeUint16
	case "int32", "dint", "long":
		return typeInt32
	case "uint32", "udint", "dword", "ulong":
		return typeUint32
	case "int64", "lint":
		return typeInt64
	case "uint64", "ulint", "ulong64":
		return typeUint64
	case "float", "float32", "real":
		return typeFloat32
	case "double", "float64", "lreal":
		return typeFloat64
	case "string", "utf8":
		return typeString
	case "bytes", "byte_array", "raw":
		return typeBytes
	default:
		return ""
	}
}

func normalizeOrder(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "big", "be", "big_endian", "big-endian":
		return orderBig
	case "little", "le", "little_endian", "little-endian":
		return orderLittle
	default:
		return ""
	}
}

func dataTypeRegisterCount(dataType string) int {
	switch dataType {
	case typeBool, typeInt8, typeUint8, typeInt16, typeUint16:
		return 1
	case typeInt32, typeUint32, typeFloat32:
		return 2
	case typeInt64, typeUint64, typeFloat64:
		return 4
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
