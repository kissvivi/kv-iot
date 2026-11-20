package tcpframe

import (
	"bytes"
	"encoding/json"
	"testing"

	"kv-iot/datacenter/protocol"
)

func TestRenderRequestTemplateAndDecodeFields(t *testing.T) {
	point := protocol.PointBinding{
		PointIdentifier: "temperature",
		DataType:        "uint16",
		Config: json.RawMessage(`{
			"requestTemplate": "AA55{value:u16le}",
			"fields": [
				{"identifier": "temperature", "offset": 2, "type": "uint16", "scale": 0.1},
				{"identifier": "enabled", "offset": 4, "type": "uint8", "bit": 1}
			]
		}`),
	}
	mapping, err := parsePointBinding(point, Framing{Type: FramingFixed, Length: 16}, 1024)
	if err != nil {
		t.Fatalf("parsePointBinding: %v", err)
	}
	request, err := buildRequest(mapping, point, uint16(0x1234))
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	if !bytes.Equal(request, []byte{0xAA, 0x55, 0x34, 0x12}) {
		t.Fatalf("request = %x", request)
	}

	values, err := decodeFields([]byte{0xAA, 0x55, 0x00, 0x7B, 0x02}, mapping.fields)
	if err != nil {
		t.Fatalf("decodeFields: %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("values = %d", len(values))
	}
	if values[0].Value != 12.3 {
		t.Fatalf("temperature = %#v", values[0].Value)
	}
	if values[1].Value != true {
		t.Fatalf("enabled = %#v", values[1].Value)
	}
}

func TestDecodeSignedAndLittleEndian(t *testing.T) {
	field, err := parseField(wireField{
		Identifier: "value",
		Type:       "int16",
		ByteOrder:  "little",
	})
	if err != nil {
		t.Fatalf("parseField: %v", err)
	}
	field.offset = 1
	values, err := decodeFields([]byte{0, 0xff, 0xff}, []fieldMapping{field})
	if err != nil {
		t.Fatalf("decodeFields: %v", err)
	}
	if values[0].Value != int64(-1) {
		t.Fatalf("value = %#v", values[0].Value)
	}
}

func TestInvalidFieldRange(t *testing.T) {
	field, err := parseField(wireField{Identifier: "value", Type: "uint32"})
	if err != nil {
		t.Fatalf("parseField: %v", err)
	}
	field.offset = 2
	if _, err := decodeFields([]byte{1, 2, 3, 4}, []fieldMapping{field}); err == nil {
		t.Fatal("expected out-of-range error")
	}
}
