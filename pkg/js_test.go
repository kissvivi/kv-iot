package pkg

import (
	"fmt"
	"testing"
)

func TestBinaryToJSON(t *testing.T) {
	type args struct {
		xxx          uint8
		variables    map[string]string
		decodeScript string
		b            []byte
	}

	decodeScript := "function Decode(){return {ss:1} };"
	a := args{decodeScript: decodeScript}
	got, err := BinaryToJSON(a.xxx, a.variables, a.decodeScript, a.b)
	fmt.Println(string(got))
	if err != nil {
		t.Errorf("BinaryToJSON() error = %v", err)
		return
	}
}

func TestBinaryToJSONInjectsBinaryData(t *testing.T) {
	got, err := BinaryToJSON(1, map[string]string{"deviceNo": "dev-001"}, "({ value: binaryData, deviceNo: deviceNo });", []byte("payload"))
	if err != nil {
		t.Fatalf("BinaryToJSON() error = %v", err)
	}

	if string(got) != `{"deviceNo":"dev-001","value":"payload"}` {
		t.Fatalf("unexpected result: %s", got)
	}
}
