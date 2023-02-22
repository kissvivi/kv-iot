package mqtt

import (
	"fmt"
	"testing"
	"time"
)

func TestMqtt_Sub(t *testing.T) {
	cfg := NewConfigMqtt("172.30.25.195", 1883, "", "", "")
	mc := NewMqtt(cfg)
	mc.Sub(1, "/#")
	for {
		select {
		case v, ok := <-SubData:
			fmt.Printf("v=%v, ok=%v\n", v, ok)
			//time.Sleep(1 * time.Second)
		default:
			fmt.Println("通道没有数据")
			time.Sleep(1 * time.Second)
		}
		fmt.Println("waiting")
	}
}
