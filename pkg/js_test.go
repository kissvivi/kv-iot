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
