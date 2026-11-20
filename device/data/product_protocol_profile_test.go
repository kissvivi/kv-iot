package data

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestProductProtocolProfileTableName(t *testing.T) {
	profile := ProductProtocolProfile{}
	if got := profile.TableName(); got != "product_protocol_profiles" {
		t.Fatalf("TableName() = %q, want %q", got, "product_protocol_profiles")
	}
}

func TestProductProtocolProfileSchema(t *testing.T) {
	parsed, err := schema.Parse(&ProductProtocolProfile{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}

	for _, column := range []string{"product_id", "protocol_type", "driver_name", "profile_version", "config_schema", "status"} {
		if parsed.LookUpField(column) == nil {
			t.Fatalf("missing column %q", column)
		}
	}

	index := parsed.LookIndex("uk_product_protocol_profile_version")
	if index == nil || index.Class != "UNIQUE" || len(index.Fields) != 2 {
		t.Fatalf("product profile version index = %#v, want unique composite index", index)
	}
	if index.Fields[0].DBName != "product_id" || index.Fields[1].DBName != "profile_version" {
		t.Fatalf("index fields = %q, %q", index.Fields[0].DBName, index.Fields[1].DBName)
	}
}
