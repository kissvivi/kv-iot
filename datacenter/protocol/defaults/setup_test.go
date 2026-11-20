package defaults

import (
	"reflect"
	"testing"

	"kv-iot/datacenter/protocol"
)

func TestNewRegistry(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	want := []string{
		protocol.ProtocolModbusTCP,
		protocol.ProtocolMQTT,
		protocol.ProtocolTCPFrame,
		protocol.ProtocolTCPRaw,
	}
	if got := registry.ProtocolTypes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ProtocolTypes() = %v, want %v", got, want)
	}
}
