package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

func main() {
	host := flag.String("host", "127.0.0.1", "Modbus TCP host")
	port := flag.Int("port", 502, "Modbus TCP port")
	start := flag.Int("start", 0, "starting address")
	quantity := flag.Int("quantity", 16, "number of values to read")
	timeout := flag.Duration("timeout", time.Second, "request timeout")
	flag.Parse()

	if *quantity < 1 || *quantity > 120 {
		fmt.Fprintln(os.Stderr, "quantity must be between 1 and 120")
		os.Exit(2)
	}

	target := net.JoinHostPort(*host, fmt.Sprintf("%d", *port))
	fmt.Printf("target=%s start=%d quantity=%d\n", target, *start, *quantity)
	for _, unitID := range []byte{0, 1, 2} {
		for _, functionCode := range []byte{1, 2, 3, 4} {
			values, err := read(target, unitID, functionCode, uint16(*start), uint16(*quantity), *timeout)
			if err != nil {
				fmt.Printf("unit=%d fc=%d error=%v\n", unitID, functionCode, err)
				continue
			}
			fmt.Printf("unit=%d fc=%d values=%v\n", unitID, functionCode, values)
		}
	}
}

func read(target string, unitID, functionCode byte, start, quantity uint16, timeout time.Duration) ([]uint16, error) {
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	request := make([]byte, 12)
	binary.BigEndian.PutUint16(request[0:2], 1)
	binary.BigEndian.PutUint16(request[2:4], 0)
	binary.BigEndian.PutUint16(request[4:6], 6)
	request[6] = unitID
	request[7] = functionCode
	binary.BigEndian.PutUint16(request[8:10], start)
	binary.BigEndian.PutUint16(request[10:12], quantity)
	if _, err := conn.Write(request); err != nil {
		return nil, err
	}

	header := make([]byte, 7)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	length := int(binary.BigEndian.Uint16(header[4:6]))
	if length < 2 || length > 254 {
		return nil, fmt.Errorf("invalid response length %d", length)
	}
	payload := make([]byte, length-1)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}
	if payload[0]&0x80 != 0 {
		return nil, fmt.Errorf("modbus exception 0x%02x", payload[1])
	}

	switch functionCode {
	case 1, 2:
		byteCount := int(payload[1])
		if byteCount < 1 || 2+byteCount > len(payload) {
			return nil, fmt.Errorf("invalid bit response")
		}
		values := make([]uint16, 0, quantity)
		for index := uint16(0); index < quantity; index++ {
			value := payload[2+index/8] & (1 << (index % 8))
			if value != 0 {
				values = append(values, 1)
			} else {
				values = append(values, 0)
			}
		}
		return values, nil
	case 3, 4:
		byteCount := int(payload[1])
		if byteCount != int(quantity)*2 || 2+byteCount > len(payload) {
			return nil, fmt.Errorf("invalid register response length %d", byteCount)
		}
		values := make([]uint16, quantity)
		for index := range values {
			values[index] = binary.BigEndian.Uint16(payload[2+index*2 : 4+index*2])
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported function code %d", functionCode)
	}
}
