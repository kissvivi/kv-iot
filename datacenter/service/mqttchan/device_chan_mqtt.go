package mqttchan

import (
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"kv-iot/datacenter/data"
	"kv-iot/datacenter/service/center"
	"kv-iot/pkg/mqtt"
	"log"
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
	// /${ProductKey}/${DeviceNo}/uplink
	d.mqttConn.Sub(1, "/+/+/uplink")
	for {
		select {
		case v, ok := <-mqtt.SubData:
			if ok {
				m := v.(MQTT.Message)
				center := center.NewCenter(d.ChannelID, m.Payload())
				msg := center.Decode()
				topic := m.Topic()
				productKey := topic
				deviceNo := topic
				deviceUserName := productKey + ":" + deviceNo
				isAuth := NewDeviceAuthsMap().Has(deviceUserName)
				msg.Device.ProductKey = productKey
				msg.Device.DeviceNo = deviceNo
				if isAuth {
					center.ToInflux(msg)
				} else {
					log.Printf("device %s is not reg...", deviceUserName)
					d.AuthDevice(msg)
				}

			}
		default:

		}
	}
}

func (d *DeviceChanMqtt) AuthDevice(msg data.KvMsg) (err error) {
	d.ChannelID = "test001"
	//验证是否注册
	//TODO 自己解析mqtt协议，做到从，mqtt的协议层解决设备注册认证问题
	// 参考阿里
	// userName:productKey+deviceNo
	// password:系统生成或者用户赋予
	log.Printf("auth device %s ing... ", msg.Device.ProductKey.(string)+":"+msg.Device.DeviceNo)
	return
}

func (d *DeviceChanMqtt) StateDevice(msg data.KvMsg) (err error) {
	fmt.Println("设备上线-----------")
	//TODO 直接推送到前端，或者消息系统代表设备上线
	return
}

func (d *DeviceChanMqtt) SendMsg(topic string, payload []byte) error {
	if d.state != ING {
		return errors.New("mqtt channel is not active")
	}
	d.mqttConn.Pub(0, topic, string(payload))
	return nil
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
