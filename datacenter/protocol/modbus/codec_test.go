package modbus

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"

	"kv-iot/datacenter/protocol"
)

func TestDecodeRegisterValues(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		registers []uint16
		want      any
	}{
		{
			name:      "uint32 big words and bytes",
			config:    `{"area":"holding_register","address":0,"dataType":"uint32"}`,
			registers: []uint16{0x1234, 0x5678},
			want:      uint32(0x12345678),
		},
		{
			name:      "uint32 little word order",
			config:    `{"area":"holding_register","address":0,"dataType":"uint32","wordOrder":"little"}`,
			registers: []uint16{0x1234, 0x5678},
			want:      uint32(0x56781234),
		},
		{
			name:      "uint32 little byte order",
			config:    `{"area":"holding_register","address":0,"dataType":"uint32","byteOrder":"little"}`,
			registers: []uint16{0x1234, 0x5678},
			want:      uint32(0x34127856),
		},
		{
			name:      "signed int16",
			config:    `{"area":"holding_register","address":0,"dataType":"int16"}`,
			registers: []uint16{0xffff},
			want:      int16(-1),
		},
		{
			name:      "float32",
			config:    `{"area":"holding_register","address":0,"dataType":"float32"}`,
			registers: []uint16{uint16(math.Float32bits(3.5) >> 16), uint16(math.Float32bits(3.5))},
			want:      float32(3.5),
		},
		{
			name:      "scale and offset",
			config:    `{"area":"holding_register","address":0,"dataType":"uint16","scale":0.1,"offset":-10}`,
			registers: []uint16{1000},
			want:      float64(90),
		},
		{
			name:      "bit offset",
			config:    `{"area":"holding_register","address":0,"dataType":"bool","bitOffset":3}`,
			registers: []uint16{0x0008},
			want:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapping := parseTestBinding(t, test.config)
			got, err := decodeValue(mapping, readResult{registers: test.registers})
			if err != nil {
				t.Fatalf("decode value: %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("value = %#v (%T), want %#v (%T)", got, got, test.want, test.want)
			}
		})
	}
}

func TestDecodeCoilValue(t *testing.T) {
	mapping := parseTestBinding(t, `{"area":"coil","address":4}`)
	got, err := decodeValue(mapping, readResult{bits: []bool{true}})
	if err != nil {
		t.Fatalf("decode coil: %v", err)
	}
	if got != true {
		t.Fatalf("coil value = %#v", got)
	}
}

func TestParsePointBindingRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config string
		target error
	}{
		{name: "unknown area", config: `{"area":"mystery","address":0}`, target: ErrInvalidBinding},
		{name: "missing address", config: `{"area":"holding_register"}`, target: ErrInvalidBinding},
		{name: "unknown type", config: `{"area":"holding_register","address":0,"dataType":"decimal128"}`, target: ErrUnsupportedType},
		{name: "short quantity", config: `{"area":"holding_register","address":0,"dataType":"float32","quantity":1}`, target: ErrInvalidBinding},
		{name: "bad byte order", config: `{"area":"holding_register","address":0,"byteOrder":"middle"}`, target: ErrInvalidBinding},
		{name: "bit offset on coil", config: `{"area":"coil","address":0,"bitOffset":1}`, target: ErrInvalidBinding},
		{name: "bit quantity", config: `{"area":"coil","address":0,"quantity":2}`, target: ErrInvalidBinding},
		{
			name:   "too many registers",
			config: `{"area":"holding_register","address":0,"dataType":"uint32","quantity":126}`,
			target: ErrInvalidBinding,
		},
		{name: "zero scale", config: `{"area":"holding_register","address":0,"scale":0}`, target: ErrInvalidBinding},
		{
			name:   "conflicting unit ids",
			config: `{"area":"holding_register","address":0,"unitId":1,"unit_id":2}`,
			target: ErrInvalidBinding,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			point := protocol.PointBinding{
				PointIdentifier: "test",
				Config:          json.RawMessage(test.config),
			}
			_, err := parsePointBinding(point, 1)
			if err == nil {
				t.Fatal("expected binding error")
			}
			if !errors.Is(err, test.target) {
				t.Fatalf("error = %v, want %v", err, test.target)
			}
		})
	}
}

func TestEncodeSingleRegister(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		value   any
		want    uint16
		wantErr bool
	}{
		{name: "uint16", config: `{"area":"holding_register","address":0,"dataType":"uint16"}`, value: 4660, want: 4660},
		{name: "int16", config: `{"area":"holding_register","address":0,"dataType":"int16"}`, value: -2, want: 0xfffe},
		{
			name:   "scale",
			config: `{"area":"holding_register","address":0,"dataType":"uint16","scale":0.1}`,
			value:  12.3,
			want:   123,
		},
		{
			name:   "byte order",
			config: `{"area":"holding_register","address":0,"dataType":"uint16","byteOrder":"little"}`,
			value:  0x1234,
			want:   0x3412,
		},
		{
			name:    "out of range",
			config:  `{"area":"holding_register","address":0,"dataType":"uint16"}`,
			value:   65536,
			wantErr: true,
		},
		{
			name:    "non integral scaled value",
			config:  `{"area":"holding_register","address":0,"dataType":"uint16","scale":0.1}`,
			value:   12.35,
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapping := parseTestBinding(t, test.config)
			got, err := encodeSingleRegister(mapping, test.value)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected encode error")
				}
				return
			}
			if err != nil {
				t.Fatalf("encode register: %v", err)
			}
			if got != test.want {
				t.Fatalf("register = 0x%04x, want 0x%04x", got, test.want)
			}
		})
	}
}

func TestExceptionMapsToSentinel(t *testing.T) {
	err := &Exception{FunctionCode: fcReadHolding, Code: 0x02}
	if !errors.Is(err, ErrIllegalAddress) {
		t.Fatalf("exception = %v, want ErrIllegalAddress", err)
	}
}

func parseTestBinding(t *testing.T, config string) pointMapping {
	t.Helper()
	mapping, err := parsePointBinding(protocol.PointBinding{
		PointIdentifier: "test",
		Config:          json.RawMessage(config),
	}, 1)
	if err != nil {
		t.Fatalf("parse binding: %v", err)
	}
	return mapping
}
