package mqtt

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestMqtt_Pub(t *testing.T) {

	//for i := 0; i < 1; i++ {
	cfg := NewConfigMqtt("172.18.61.43", 1883, "", "", "")
	mc := NewMqtt(cfg)
	for j := 0; j < 1000; j++ {
		time.Sleep(5 * time.Second)
		val := rand.Intn(40)
		mc.Pub(1, "温度计/TMP-001/uplink", fmt.Sprintf("{\n  \"product_key\": \"温度计\",\n  \"property\":{\n    \"name\":\"温度\",\n    \"identifier\":\"温度\",\n    \"value\":%d\n  }\n}", val))
	}
	//}

	//time.Sleep(time.Minute)
}
