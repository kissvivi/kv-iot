package mqttbroker

import (
	"errors"
	"fmt"
	"io"
)

const (
	packetConnect     byte = 1
	packetConnAck     byte = 2
	packetPublish     byte = 3
	packetPubAck      byte = 4
	packetSubscribe   byte = 8
	packetSubAck      byte = 9
	packetUnsubscribe byte = 10
	packetUnsubAck    byte = 11
	packetPingReq     byte = 12
	packetPingResp    byte = 13
	packetDisconnect  byte = 14
)

type packet struct {
	header  byte
	payload []byte
}

func readPacket(r io.Reader, maxPacketSize int) (packet, error) {
	var first [1]byte
	if _, err := io.ReadFull(r, first[:]); err != nil {
		return packet{}, err
	}
	remaining, err := readRemainingLength(r)
	if err != nil {
		return packet{}, err
	}
	if maxPacketSize > 0 && remaining > maxPacketSize {
		return packet{}, fmt.Errorf("mqttbroker: packet size %d exceeds limit %d", remaining, maxPacketSize)
	}
	payload := make([]byte, remaining)
	if _, err := io.ReadFull(r, payload); err != nil {
		return packet{}, err
	}
	return packet{header: first[0], payload: payload}, nil
}

func writePacket(w io.Writer, header byte, payload []byte) error {
	packet := make([]byte, 1, 1+4+len(payload))
	packet[0] = header
	packet = appendRemainingLength(packet, len(payload))
	packet = append(packet, payload...)
	for len(packet) > 0 {
		n, err := w.Write(packet)
		if err != nil {
			return err
		}
		packet = packet[n:]
	}
	return nil
}

func readRemainingLength(r io.Reader) (int, error) {
	multiplier := 1
	value := 0
	for i := 0; i < 4; i++ {
		var encoded [1]byte
		if _, err := io.ReadFull(r, encoded[:]); err != nil {
			return 0, err
		}
		value += int(encoded[0]&0x7F) * multiplier
		if encoded[0]&0x80 == 0 {
			return value, nil
		}
		multiplier *= 128
	}
	return 0, errors.New("mqttbroker: malformed remaining length")
}

func appendRemainingLength(dst []byte, length int) []byte {
	for {
		encoded := byte(length % 128)
		length /= 128
		if length > 0 {
			encoded |= 0x80
		}
		dst = append(dst, encoded)
		if length == 0 {
			return dst
		}
	}
}

func appendMQTTString(dst []byte, value string) []byte {
	dst = append(dst, byte(len(value)>>8), byte(len(value)))
	return append(dst, value...)
}

func readMQTTString(payload []byte, offset *int) (string, error) {
	if *offset+2 > len(payload) {
		return "", errors.New("mqttbroker: truncated string length")
	}
	length := int(payload[*offset])<<8 | int(payload[*offset+1])
	*offset += 2
	if *offset+length > len(payload) {
		return "", errors.New("mqttbroker: truncated string")
	}
	value := string(payload[*offset : *offset+length])
	*offset += length
	return value, nil
}
