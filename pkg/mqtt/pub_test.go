package mqtt

import (
	"testing"
)

func TestMqtt_Pub(t *testing.T) {

	cfg := NewConfigMqtt("172.30.25.195", 1883, "", "", "")
	mc := NewMqtt(cfg)

	mc.Pub(1, "/test/test", "11111111111111111111")
	//time.Sleep(time.Minute)
}
