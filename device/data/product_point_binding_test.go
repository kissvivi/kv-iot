package data

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestProductPointBindingTableName(t *testing.T) {
	binding := ProductPointBinding{}
	if got := binding.TableName(); got != "product_point_bindings" {
		t.Fatalf("TableName() = %q, want %q", got, "product_point_bindings")
	}
}

func TestProductPointBindingSchema(t *testing.T) {
	parsed, err := schema.Parse(&ProductPointBinding{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}

	index := parsed.LookIndex("uk_product_point_binding_point")
	if index == nil || index.Class != "UNIQUE" || len(index.Fields) != 3 {
		t.Fatalf("product point binding index = %#v, want unique composite index", index)
	}
	wantFields := []string{"profile_id", "point_kind", "point_id"}
	for i, want := range wantFields {
		if index.Fields[i].DBName != want {
			t.Fatalf("index field %d = %q, want %q", i, index.Fields[i].DBName, want)
		}
	}
}
