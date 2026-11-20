package tcpframe

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

func (f Framing) readFrame(conn net.Conn, maxFrameSize int) ([]byte, error) {
	switch f.Type {
	case FramingFixed:
		frame := make([]byte, f.Length)
		if _, err := io.ReadFull(conn, frame); err != nil {
			return nil, err
		}
		return frame, nil
	case FramingLength:
		headerLength := f.LengthOffset + f.LengthSize
		header := make([]byte, headerLength)
		if _, err := io.ReadFull(conn, header); err != nil {
			return nil, err
		}
		lengthValue, err := decodeLength(header[f.LengthOffset:], f.LengthSize, f.Endian)
		if err != nil {
			return nil, err
		}
		totalLength := int(lengthValue) + headerLength
		if f.LengthIncludesHeader {
			totalLength = int(lengthValue)
		}
		if totalLength < headerLength {
			return nil, fmt.Errorf("%w: length %d is smaller than header %d", ErrInvalidFrame, totalLength, headerLength)
		}
		if totalLength > maxFrameSize {
			return nil, fmt.Errorf("%w: length %d exceeds %d", ErrFrameTooLarge, totalLength, maxFrameSize)
		}
		frame := make([]byte, totalLength)
		copy(frame, header)
		if _, err := io.ReadFull(conn, frame[headerLength:]); err != nil {
			return nil, err
		}
		return frame, nil
	case FramingDelimiter:
		frame := make([]byte, 0, 64)
		one := make([]byte, 1)
		for len(frame) < maxFrameSize {
			if _, err := io.ReadFull(conn, one); err != nil {
				return nil, err
			}
			frame = append(frame, one[0])
			if bytes.HasSuffix(frame, f.Delimiter) {
				return frame, nil
			}
		}
		return nil, fmt.Errorf("%w: delimiter not found within %d bytes", ErrFrameTooLarge, maxFrameSize)
	default:
		return nil, fmt.Errorf("%w: unsupported framing %q", ErrInvalidFrame, f.Type)
	}
}

func (f Framing) encodePayload(payload []byte, maxFrameSize int) ([]byte, error) {
	switch f.Type {
	case FramingFixed:
		if len(payload) != f.Length {
			return nil, fmt.Errorf("%w: fixed framing requires %d bytes, got %d", ErrInvalidFrame, f.Length, len(payload))
		}
		return append([]byte(nil), payload...), nil
	case FramingLength:
		headerLength := f.LengthOffset + f.LengthSize
		totalLength := headerLength + len(payload)
		if totalLength > maxFrameSize {
			return nil, fmt.Errorf("%w: request length %d exceeds %d", ErrFrameTooLarge, totalLength, maxFrameSize)
		}
		lengthValue := len(payload)
		if f.LengthIncludesHeader {
			lengthValue = totalLength
		}
		frame := make([]byte, totalLength)
		if err := encodeLength(frame[f.LengthOffset:f.LengthOffset+f.LengthSize], uint32(lengthValue), f.Endian); err != nil {
			return nil, err
		}
		copy(frame[headerLength:], payload)
		return frame, nil
	case FramingDelimiter:
		frame := append([]byte(nil), payload...)
		if !bytes.HasSuffix(frame, f.Delimiter) {
			frame = append(frame, f.Delimiter...)
		}
		if len(frame) > maxFrameSize {
			return nil, fmt.Errorf("%w: request length %d exceeds %d", ErrFrameTooLarge, len(frame), maxFrameSize)
		}
		return frame, nil
	default:
		return nil, fmt.Errorf("%w: unsupported framing %q", ErrInvalidFrame, f.Type)
	}
}

func decodeLength(data []byte, size int, endian string) (uint32, error) {
	switch size {
	case 1:
		return uint32(data[0]), nil
	case 2:
		if endian == "little" {
			return uint32(binary.LittleEndian.Uint16(data)), nil
		}
		return uint32(binary.BigEndian.Uint16(data)), nil
	case 4:
		if endian == "little" {
			return binary.LittleEndian.Uint32(data), nil
		}
		return binary.BigEndian.Uint32(data), nil
	default:
		return 0, fmt.Errorf("%w: unsupported length size %d", ErrInvalidFrame, size)
	}
}

func encodeLength(target []byte, value uint32, endian string) error {
	switch len(target) {
	case 1:
		if value > 0xff {
			return fmt.Errorf("%w: length %d does not fit in one byte", ErrInvalidFrame, value)
		}
		target[0] = byte(value)
	case 2:
		if value > 0xffff {
			return fmt.Errorf("%w: length %d does not fit in two bytes", ErrInvalidFrame, value)
		}
		if endian == "little" {
			binary.LittleEndian.PutUint16(target, uint16(value))
		} else {
			binary.BigEndian.PutUint16(target, uint16(value))
		}
	case 4:
		if endian == "little" {
			binary.LittleEndian.PutUint32(target, value)
		} else {
			binary.BigEndian.PutUint32(target, value)
		}
	default:
		return fmt.Errorf("%w: unsupported length size %d", ErrInvalidFrame, len(target))
	}
	return nil
}
