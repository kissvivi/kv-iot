package tcpframe

import (
	"bytes"
	"errors"
	"testing"
)

func TestCRC16Modbus(t *testing.T) {
	value := crc16Modbus([]byte("123456789"))
	if value != 0x4b37 {
		t.Fatalf("crc = %04x, want 4b37", value)
	}
	payload, err := appendChecksum([]byte("123456789"), ChecksumCRC16Modbus)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	decoded, err := verifyChecksum(payload, ChecksumCRC16Modbus)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if string(decoded) != "123456789" {
		t.Fatalf("decoded = %q", decoded)
	}
}

func TestSumAndXORChecksums(t *testing.T) {
	sum, err := appendChecksum([]byte{1, 2, 3}, ChecksumSum8)
	if err != nil {
		t.Fatalf("sum: %v", err)
	}
	if !bytes.Equal(sum, []byte{1, 2, 3, 6}) {
		t.Fatalf("sum = %x", sum)
	}
	xor, err := appendChecksum([]byte{1, 2, 3}, ChecksumXOR8)
	if err != nil {
		t.Fatalf("xor: %v", err)
	}
	if !bytes.Equal(xor, []byte{1, 2, 3, 0}) {
		t.Fatalf("xor = %x", xor)
	}
}

func TestChecksumMismatch(t *testing.T) {
	_, err := verifyChecksum([]byte{1, 2, 0}, ChecksumSum8)
	if !errors.Is(err, ErrChecksum) {
		t.Fatalf("error = %v, want ErrChecksum", err)
	}
}
