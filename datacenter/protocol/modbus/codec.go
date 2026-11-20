package modbus

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
)

type readResult struct {
	bits      []bool
	registers []uint16
}

func decodeValue(mapping pointMapping, result readResult) (any, error) {
	if mapping.area == areaCoil || mapping.area == areaDiscreteInput {
		if len(result.bits) == 0 {
			return nil, fmt.Errorf("%w: empty bit response", ErrInvalidResponse)
		}
		return result.bits[0], nil
	}
	if len(result.registers) < mapping.quantity {
		return nil, fmt.Errorf("%w: register response has %d words, expected %d",
			ErrInvalidResponse, len(result.registers), mapping.quantity)
	}
	if mapping.bitOffset != nil {
		word := result.registers[0]
		return word&(uint16(1)<<uint(*mapping.bitOffset)) != 0, nil
	}

	data := registersToBytes(result.registers[:mapping.quantity], mapping.byteOrder, mapping.wordOrder)
	value, err := decodeRegisters(data, mapping.dataType)
	if err != nil {
		return nil, err
	}
	return applyScale(value, mapping.scale, mapping.offset)
}

func registersToBytes(registers []uint16, byteOrder, wordOrder string) []byte {
	ordered := append([]uint16(nil), registers...)
	if wordOrder == orderLittle {
		for left, right := 0, len(ordered)-1; left < right; left, right = left+1, right-1 {
			ordered[left], ordered[right] = ordered[right], ordered[left]
		}
	}

	data := make([]byte, len(ordered)*2)
	for index, word := range ordered {
		data[index*2] = byte(word >> 8)
		data[index*2+1] = byte(word)
		if byteOrder == orderLittle {
			data[index*2], data[index*2+1] = data[index*2+1], data[index*2]
		}
	}
	return data
}

func bytesToRegisters(data []byte, byteOrder, wordOrder string) ([]uint16, error) {
	if len(data) == 0 || len(data)%2 != 0 {
		return nil, fmt.Errorf("%w: register data must contain complete 16-bit words", ErrInvalidBinding)
	}
	registers := make([]uint16, len(data)/2)
	for index := range registers {
		high := data[index*2]
		low := data[index*2+1]
		if byteOrder == orderLittle {
			high, low = low, high
		}
		registers[index] = binary.BigEndian.Uint16([]byte{high, low})
	}
	if wordOrder == orderLittle {
		for left, right := 0, len(registers)-1; left < right; left, right = left+1, right-1 {
			registers[left], registers[right] = registers[right], registers[left]
		}
	}
	return registers, nil
}

func decodeRegisters(data []byte, dataType string) (any, error) {
	switch dataType {
	case typeBool:
		if len(data) < 2 {
			return nil, fmt.Errorf("%w: bool requires one register", ErrInvalidResponse)
		}
		return binary.BigEndian.Uint16(data[:2]) != 0, nil
	case typeUint8:
		if len(data) < 1 {
			return nil, fmt.Errorf("%w: uint8 requires one byte", ErrInvalidResponse)
		}
		return data[len(data)-1], nil
	case typeInt8:
		if len(data) < 1 {
			return nil, fmt.Errorf("%w: int8 requires one byte", ErrInvalidResponse)
		}
		return int8(data[len(data)-1]), nil
	case typeUint16:
		if len(data) < 2 {
			return nil, fmt.Errorf("%w: uint16 requires two bytes", ErrInvalidResponse)
		}
		return binary.BigEndian.Uint16(data[:2]), nil
	case typeInt16:
		if len(data) < 2 {
			return nil, fmt.Errorf("%w: int16 requires two bytes", ErrInvalidResponse)
		}
		return int16(binary.BigEndian.Uint16(data[:2])), nil
	case typeUint32:
		if len(data) < 4 {
			return nil, fmt.Errorf("%w: uint32 requires four bytes", ErrInvalidResponse)
		}
		return binary.BigEndian.Uint32(data[:4]), nil
	case typeInt32:
		if len(data) < 4 {
			return nil, fmt.Errorf("%w: int32 requires four bytes", ErrInvalidResponse)
		}
		return int32(binary.BigEndian.Uint32(data[:4])), nil
	case typeUint64:
		if len(data) < 8 {
			return nil, fmt.Errorf("%w: uint64 requires eight bytes", ErrInvalidResponse)
		}
		return binary.BigEndian.Uint64(data[:8]), nil
	case typeInt64:
		if len(data) < 8 {
			return nil, fmt.Errorf("%w: int64 requires eight bytes", ErrInvalidResponse)
		}
		return int64(binary.BigEndian.Uint64(data[:8])), nil
	case typeFloat32:
		if len(data) < 4 {
			return nil, fmt.Errorf("%w: float32 requires four bytes", ErrInvalidResponse)
		}
		return math.Float32frombits(binary.BigEndian.Uint32(data[:4])), nil
	case typeFloat64:
		if len(data) < 8 {
			return nil, fmt.Errorf("%w: float64 requires eight bytes", ErrInvalidResponse)
		}
		return math.Float64frombits(binary.BigEndian.Uint64(data[:8])), nil
	case typeString:
		return string(data), nil
	case typeBytes:
		return append([]byte(nil), data...), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, dataType)
	}
}

func applyScale(value any, scale, offset float64) (any, error) {
	if scale == 1 && offset == 0 {
		return value, nil
	}
	numeric, ok := numericValue(value)
	if !ok {
		if _, isBool := value.(bool); isBool {
			return value, nil
		}
		return nil, fmt.Errorf("%w: scale/offset require a numeric value", ErrUnsupportedType)
	}
	return numeric*scale + offset, nil
}

func encodeSingleRegister(mapping pointMapping, value any) (uint16, error) {
	if mapping.bitOffset != nil {
		return 0, fmt.Errorf("%w: bitOffset writes require read-modify-write", ErrInvalidWriteValue)
	}
	if mapping.quantity != 1 || mapping.registerLen != 1 {
		return 0, fmt.Errorf("%w: single-register writes require one register", ErrInvalidWriteValue)
	}

	if mapping.dataType == typeBool {
		boolean, err := boolValue(value)
		if err != nil {
			return 0, err
		}
		if boolean {
			return 1, nil
		}
		return 0, nil
	}

	numeric, err := numericValueStrict(value)
	if err != nil {
		return 0, err
	}
	raw := (numeric - mapping.offset) / mapping.scale
	if math.IsNaN(raw) || math.IsInf(raw, 0) {
		return 0, fmt.Errorf("%w: value is not finite", ErrInvalidWriteValue)
	}
	rounded := math.Round(raw)
	if math.Abs(raw-rounded) > 1e-9 {
		return 0, fmt.Errorf("%w: value does not map to an integer register", ErrInvalidWriteValue)
	}

	var register uint16
	switch mapping.dataType {
	case typeInt8:
		if rounded < math.MinInt8 || rounded > math.MaxInt8 {
			return 0, fmt.Errorf("%w: int8 out of range", ErrInvalidWriteValue)
		}
		register = uint16(int16(int8(rounded)))
	case typeUint8:
		if rounded < 0 || rounded > math.MaxUint8 {
			return 0, fmt.Errorf("%w: uint8 out of range", ErrInvalidWriteValue)
		}
		register = uint16(uint8(rounded))
	case typeInt16:
		if rounded < math.MinInt16 || rounded > math.MaxInt16 {
			return 0, fmt.Errorf("%w: int16 out of range", ErrInvalidWriteValue)
		}
		register = uint16(int16(rounded))
	case typeUint16:
		if rounded < 0 || rounded > math.MaxUint16 {
			return 0, fmt.Errorf("%w: uint16 out of range", ErrInvalidWriteValue)
		}
		register = uint16(rounded)
	default:
		return 0, fmt.Errorf("%w: single-register write does not support %s", ErrInvalidWriteValue, mapping.dataType)
	}

	if mapping.byteOrder == orderLittle {
		register = register<<8 | register>>8
	}
	return register, nil
}

func bitWriteValue(value any) (bool, error) {
	return boolValue(value)
}

func boolValue(value any) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		switch typed {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		}
	}
	numeric, ok := numericValue(value)
	if !ok || (numeric != 0 && numeric != 1) {
		return false, fmt.Errorf("%w: expected bool or 0/1", ErrInvalidWriteValue)
	}
	return numeric == 1, nil
}

func numericValueStrict(value any) (float64, error) {
	numeric, ok := numericValue(value)
	if !ok {
		return 0, fmt.Errorf("%w: expected a numeric value", ErrInvalidWriteValue)
	}
	if math.IsNaN(numeric) || math.IsInf(numeric, 0) {
		return 0, fmt.Errorf("%w: value must be finite", ErrInvalidWriteValue)
	}
	return numeric, nil
}

func numericValue(value any) (float64, bool) {
	if value == nil {
		return 0, false
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(reflected.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(reflected.Uint()), true
	case reflect.Float32, reflect.Float64:
		return reflected.Float(), true
	default:
		return 0, false
	}
}
