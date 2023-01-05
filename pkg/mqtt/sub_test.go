package mqtt

import (
	"testing"
)

func TestMqtt_Sub(t *testing.T) {
	cfg := NewConfigMqtt("172.30.25.195", 1883, "", "", "")
	mc := NewMqtt(cfg)

	for {
		mc.Sub(1, "/test/test")
	}
}
