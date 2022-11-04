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

	decodeScript := "function Decode(ss){ss = 1 ;console.log(ss) };"
	a := args{decodeScript: decodeScript}
	got, err := BinaryToJSON(a.xxx, a.variables, a.decodeScript, a.b)
	fmt.Println(got)
	if err != nil {
		t.Errorf("BinaryToJSON() error = %v", err)
		return
	}
}
