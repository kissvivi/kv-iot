package mq

import (
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"kv-iot/datacenter/service/center"
	"kv-iot/pkg/mqtt"
)

func SSub() {
	cfg := mqtt.NewConfigMqtt("172.19.77.116", 1883, "", "", "")
	mc := mqtt.NewMqtt(cfg)
	mc.Sub(1, "/test/test")
	for {

		select {
		case v, ok := <-mqtt.SubData:
			fmt.Printf("+v=%+v, ok=%v\n", v, ok)
			m := v.(MQTT.Message)
			fmt.Println(m.MessageID())
			center := center.NewCenter("mqtt-01", m.Payload())
			center.ToInflux(center.Decode())
			//time.Sleep(1 * time.Second)
		default:
			//fmt.Println("监听数据通道，通道没有数据")
			//time.Sleep(1 * time.Second)
		}
		//fmt.Println("waiting")
	}
}
