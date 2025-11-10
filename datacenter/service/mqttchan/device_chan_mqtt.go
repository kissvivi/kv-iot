package mqttchan

import (
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"kv-iot/config"
	"kv-iot/datacenter/data"
	"kv-iot/datacenter/service/center"
	"kv-iot/datacenter/service/device_auth"
	"kv-iot/pkg/mqtt"
	"log"
	"strconv"
	"sync"
)

const (
	CLOSE = iota
	ING
	DISABLE
)

// DeviceChanMqtt 用于处理从设备发送过来的数据，然后注册设备并进行转发
// 内部实现是使用MQTT协议，主要功能包括：
// 1.建立与MQTT服务器的连接
// 2.订阅设备主题
// 3.处理设备注册和数据传输
// 4.优雅关闭连接

type DeviceChanMqtt struct {
	mqttConn  *mqtt.Mqtt
	state     int
	ChannelID string
	data.DeviceChanConfig
	running bool
	mu      sync.RWMutex
	wg      sync.WaitGroup
}

// NewDeviceChanMqtt 创建一个新的DeviceChanMqtt实例
func NewDeviceChanMqtt() *DeviceChanMqtt {
	return &DeviceChanMqtt{
		running: true,
	}
}

// Create 创建MQTT连接并订阅主题
func (d *DeviceChanMqtt) Create() error {
	// 从配置文件读取MQTT配置
	mqttCfg := config.CONFIG.Datasource.Mqtt
	port, err := strconv.Atoi(mqttCfg.Port)
	if err != nil {
		log.Printf("MQTT端口配置无效，使用默认值1883: %v", err)
		port = 1883
	}

	// 使用配置创建MQTT客户端
	cfg := mqtt.NewConfigMqtt(mqttCfg.Url, port, mqttCfg.ClientID, mqttCfg.Username, mqttCfg.Password)
	log.Printf("正在连接MQTT服务器: %s:%d, 客户端ID: %s", mqttCfg.Url, port, mqttCfg.ClientID)
	mc := mqtt.NewMqtt(cfg)
	d.mqttConn = mc
	d.state = ING

	// 订阅设备主题
	d.mqttConn.Sub(1, "/+/+/uplink")
	log.Println("已订阅设备主题 /+/+/uplink")

	log.Println("MQTT通道初始化成功")
	return nil
}

// RegDevice 处理设备注册和消息
func (d *DeviceChanMqtt) RegDevice() {
	d.wg.Add(1)
	defer d.wg.Done()

	log.Println("开始处理设备消息")

	for {
		// 检查是否继续运行
		d.mu.RLock()
		running := d.running
		d.mu.RUnlock()

		if !running {
			log.Println("消息处理循环已停止")
			break
		}

		// 从MQTT客户端接收消息
		select {
		case v, ok := <-mqtt.SubData:
			if !ok {
				log.Println("MQTT订阅通道已关闭")
				return
			}

			m := v.(MQTT.Message)
			log.Printf("接收到MQTT消息: 主题=%s, 有效载荷长度=%d", m.Topic(), len(m.Payload()))

			center := center.NewCenter(d.ChannelID, m.Payload())
			msg := center.Decode()
			topic := m.Topic()
			productKey := topic
			deviceNo := topic
			deviceUserName := productKey + ":" + deviceNo

			// 认证设备
			isAuth := device_auth.NewDeviceAuthsMap().Has(deviceUserName)
			msg.Device.ProductKey = productKey
			msg.Device.DeviceNo = deviceNo

			if isAuth {
				// 存储数据
				if err := center.ToInflux(msg); err != nil {
					log.Printf("InfluxDB存储失败: %v", err)
				}
			} else {
				log.Printf("device %s is not reg...", deviceUserName)
				d.AuthDevice(msg)
			}
		default:
			// 防止CPU占用过高
			if !d.running {
				return
			}
		}
	}
}

// AuthDevice 认证设备
func (d *DeviceChanMqtt) AuthDevice(msg data.KvMsg) (err error) {
	d.ChannelID = "test001"
	//验证是否注册
	//TODO 自己解析mqtt协议，做到从，mqtt的协议层解决设备注册认证问题
	// 参考阿里
	// userName:productKey+deviceNo
	// password:系统生成或者用户赋予
	deviceUserName := msg.Device.ProductKey.(string) + ":" + msg.Device.DeviceNo
	log.Printf("auth device %s ing... ", deviceUserName)
	return
}

// StateDevice 设备状态处理
func (d *DeviceChanMqtt) StateDevice(msg data.KvMsg) (err error) {
	fmt.Println("设备上线-----------")
	//TODO 直接推送到前端，或者消息系统代表设备上线
	return
}

// SendMsg 向设备发送消息
func (d *DeviceChanMqtt) SendMsg(topic string, payload []byte) error {
	if d.state != ING {
		return errors.New("mqtt channel is not active")
	}
	d.mqttConn.Pub(0, topic, string(payload))
	return nil
}

// Disable 禁用MQTT通道
func (d *DeviceChanMqtt) Disable() {
	d.mu.Lock()
	d.state = DISABLE
	d.running = false
	d.mu.Unlock()
}

// Close 优雅关闭MQTT通道
func (d *DeviceChanMqtt) Close() {
	log.Println("开始关闭MQTT通道...")

	// 设置运行状态为false
	d.mu.Lock()
	d.state = CLOSE
	d.running = false
	d.mu.Unlock()

	// 等待消息处理协程退出
	go func() {
		d.wg.Wait()
		log.Println("消息处理协程已退出")
	}()

	// 关闭MQTT连接 - 直接使用底层MQTT客户端连接
	if d.mqttConn != nil && d.mqttConn.Conn != nil {
		// 使用paho.mqtt.golang客户端的Disconnect方法，参数为断开前等待的毫秒数
		d.mqttConn.Conn.Disconnect(250)
		log.Println("MQTT连接已断开")
	}

	log.Println("MQTT通道关闭完成")
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
