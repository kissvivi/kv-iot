package tcpframe

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	ChecksumNone        = "none"
	ChecksumCRC16Modbus = "crc16_modbus"
	ChecksumCRC16CCITT  = "crc16_ccitt"
	ChecksumSum8        = "sum8"
	ChecksumXOR8        = "xor8"
)

func normalizeChecksum(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none", "off", "disabled":
		return ChecksumNone, nil
	case "crc16", "crc16_modbus", "modbus_crc16", "crc16-modbus":
		return ChecksumCRC16Modbus, nil
	case "crc16_ccitt", "crc16-ccitt", "ccitt":
		return ChecksumCRC16CCITT, nil
	case "sum8", "sum", "checksum8":
		return ChecksumSum8, nil
	case "xor8", "xor":
		return ChecksumXOR8, nil
	default:
		return "", fmt.Errorf("%w: unsupported checksum %q", ErrInvalidBinding, value)
	}
}

func appendChecksum(payload []byte, checksum string) ([]byte, error) {
	algorithm, err := normalizeChecksum(checksum)
	if err != nil {
		return nil, err
	}
	result := append([]byte(nil), payload...)
	switch algorithm {
	case ChecksumNone:
		return result, nil
	case ChecksumCRC16Modbus:
		value := crc16Modbus(payload)
		return append(result, byte(value), byte(value>>8)), nil
	case ChecksumCRC16CCITT:
		value := crc16CCITT(payload)
		return append(result, byte(value>>8), byte(value)), nil
	case ChecksumSum8:
		var value byte
		for _, item := range payload {
			value += item
		}
		return append(result, value), nil
	case ChecksumXOR8:
		var value byte
		for _, item := range payload {
			value ^= item
		}
		return append(result, value), nil
	default:
		return nil, fmt.Errorf("%w: unsupported checksum %q", ErrInvalidBinding, checksum)
	}
}

func verifyChecksum(frame []byte, checksum string) ([]byte, error) {
	algorithm, err := normalizeChecksum(checksum)
	if err != nil {
		return nil, err
	}
	checksumSize := 0
	switch algorithm {
	case ChecksumNone:
		return append([]byte(nil), frame...), nil
	case ChecksumCRC16Modbus, ChecksumCRC16CCITT:
		checksumSize = 2
	case ChecksumSum8, ChecksumXOR8:
		checksumSize = 1
	}
	if len(frame) < checksumSize {
		return nil, fmt.Errorf("%w: frame is shorter than checksum", ErrInvalidFrame)
	}
	payload := frame[:len(frame)-checksumSize]
	expected, err := appendChecksum(payload, algorithm)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(expected[len(payload):], frame[len(payload):]) {
		return nil, fmt.Errorf("%w: %s", ErrChecksum, algorithm)
	}
	return append([]byte(nil), payload...), nil
}

func crc16Modbus(data []byte) uint16 {
	crc := uint16(0xffff)
	for _, item := range data {
		crc ^= uint16(item)
		for bit := 0; bit < 8; bit++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xa001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

func crc16CCITT(data []byte) uint16 {
	crc := uint16(0xffff)
	for _, item := range data {
		crc ^= uint16(item) << 8
		for bit := 0; bit < 8; bit++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
