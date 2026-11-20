package service

import (
	"testing"

	"gorm.io/gorm"

	"kv-iot/device/data"
)

func TestMissingProductBindings(t *testing.T) {
	productBindings := []data.ProductPointBinding{
		{Model: gorm.Model{ID: 1}, Enabled: true},
		{Model: gorm.Model{ID: 2}, Enabled: false},
		{Model: gorm.Model{ID: 3}, Enabled: true},
	}
	existing := []data.DevicePointBinding{
		{ProductPointBindingID: 1, ChannelInstanceID: 10},
		{ProductPointBindingID: 3, ChannelInstanceID: 9},
	}
	missing := missingProductBindings(productBindings, existing, 10)
	if len(missing) != 1 || missing[0].ID != 3 {
		t.Fatalf("missing bindings = %#v, want only binding 3", missing)
	}
}
