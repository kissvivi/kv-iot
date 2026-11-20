package service

import (
	"strings"
	"testing"

	"kv-iot/device/data"
)

func TestValidateProductModel(t *testing.T) {
	minValue := 0.0
	maxValue := 100.0
	model := ProductModel{
		Properties: []data.KvProperty{
			{
				Name:       "Temperature",
				Identifier: "temperature",
				DataType:   "float64",
				ReportMode: "interval",
				MinValue:   &minValue,
				MaxValue:   &maxValue,
			},
		},
		Actions: []data.KvAction{
			{Name: "Set threshold", Identifier: "set_threshold"},
		},
		Events: []data.KvEvent{
			{Name: "Overheat", Identifier: "overheat"},
		},
	}
	if err := validateProductModel(&model); err != nil {
		t.Fatalf("validateProductModel() error = %v", err)
	}
	if model.Actions[0].TimeoutMs != 3000 {
		t.Fatalf("action timeout = %d, want 3000", model.Actions[0].TimeoutMs)
	}
	if model.Events[0].Severity != "info" {
		t.Fatalf("event severity = %q, want info", model.Events[0].Severity)
	}
}

func TestValidateProductModelRejectsDuplicateIdentifier(t *testing.T) {
	model := ProductModel{
		Properties: []data.KvProperty{
			{Name: "A", Identifier: "value", DataType: "int32"},
			{Name: "B", Identifier: "value", DataType: "int32"},
		},
	}
	if err := validateProductModel(&model); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("validateProductModel() error = %v, want duplicate identifier", err)
	}
}

func TestValidatePropertyValue(t *testing.T) {
	property := data.KvProperty{Identifier: "temperature", DataType: "float64"}
	if err := validatePropertyValue(property, 23.5); err != nil {
		t.Fatalf("numeric property rejected: %v", err)
	}
	if err := validatePropertyValue(property, "23.5"); err == nil {
		t.Fatal("string value accepted for numeric property")
	}

	boolean := data.KvProperty{Identifier: "enabled", DataType: "bool"}
	if err := validatePropertyValue(boolean, true); err != nil {
		t.Fatalf("boolean property rejected: %v", err)
	}
}
