package data

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestDevicePointBindingTableName(t *testing.T) {
	binding := DevicePointBinding{}
	if got := binding.TableName(); got != "device_point_bindings" {
		t.Fatalf("TableName() = %q, want %q", got, "device_point_bindings")
	}
}

func TestDevicePointBindingSchema(t *testing.T) {
	parsed, err := schema.Parse(&DevicePointBinding{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}

	index := parsed.LookIndex("uk_device_point_binding_instance")
	if index == nil || index.Class != "UNIQUE" || len(index.Fields) != 3 {
		t.Fatalf("device point binding index = %#v, want unique composite index", index)
	}
	wantFields := []string{"device_id", "product_point_binding_id", "channel_instance_id"}
	for i, want := range wantFields {
		if index.Fields[i].DBName != want {
			t.Fatalf("index field %d = %q, want %q", i, index.Fields[i].DBName, want)
		}
	}
}
