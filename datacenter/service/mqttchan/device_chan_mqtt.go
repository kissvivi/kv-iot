package mqttchan

import (
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"kv-iot/datacenter/data"
	"kv-iot/datacenter/service/center"
	"kv-iot/pkg/mqtt"
)

const (
	CLOSE = iota
	ING
	DISABLE
)

type DeviceChanMqtt struct {
	mqttConn  *mqtt.Mqtt
	state     int
	ChannelID string
	data.DeviceChanConfig
}

func (d *DeviceChanMqtt) RegDevice() {
	d.mqttConn.Sub(1, "/test/test")
	for {
		select {
		case v, ok := <-mqtt.SubData:
			if ok {
				m := v.(MQTT.Message)
				center := center.NewCenter(d.ChannelID, m.Payload())
				center.ToInflux(center.Decode())
			}
		default:

		}
	}
}

func (d *DeviceChanMqtt) AuthDevice() {
	d.ChannelID = "test001"
}

func (d *DeviceChanMqtt) StateDevice() {
	fmt.Println("设备上线-----------")
}

func (d *DeviceChanMqtt) SendMsg() {
	//TODO implement me
	panic("implement me")
}

func NewDeviceChanMqtt() *DeviceChanMqtt {
	return &DeviceChanMqtt{}
}

func (d *DeviceChanMqtt) Create() {
	cfg := mqtt.NewConfigMqtt(d.Ip, d.Port, "", "", "")
	mc := mqtt.NewMqtt(cfg)
	d.mqttConn = mc
	d.state = ING
}

func (d *DeviceChanMqtt) Disable() {
	d.state = DISABLE
}

func (d *DeviceChanMqtt) Close() {
	d.state = CLOSE
}

func (d *DeviceChanMqtt) name() {

}

func SSub() {
	cfg := mqtt.NewConfigMqtt("172.19.77.116", 1883, "", "", "")
	mc := mqtt.NewMqtt(cfg)
	mc.Sub(1, "/test/test")
	for {
		select {
		case v, ok := <-mqtt.SubData:
			if ok {
				//fmt.Printf("+v=%+v, ok=%v\n", v, ok)
				m := v.(MQTT.Message)
				//fmt.Println(m.MessageID())
				center := center.NewCenter("mqttchan-01", m.Payload())
				center.ToInflux(center.Decode())
				//time.Sleep(1 * time.Second)
			}
		default:
			//fmt.Println("监听数据通道，通道没有数据")
			//time.Sleep(1 * time.Second)
		}
		//fmt.Println("waiting")
	}
}
