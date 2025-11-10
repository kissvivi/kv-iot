package mqttchan

import (
	"fmt"
	"strconv"
	"strings"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"kv-iot/config"
	"kv-iot/datacenter/data"
	"kv-iot/datacenter/service/center"
	"kv-iot/datacenter/service/device_auth"
	"kv-iot/pkg/mqtt"
	"log"
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
	running   bool
	mu        sync.RWMutex
	wg        sync.WaitGroup
	authMap   *device_auth.DeviceAuthsMap
}

// NewDeviceChanMqtt 创建一个新的DeviceChanMqtt实例
func NewDeviceChanMqtt() *DeviceChanMqtt {
	return &DeviceChanMqtt{
		running: true,
		state:     CLOSE,
		ChannelID: "mqtt-default",
		authMap:   device_auth.NewDeviceAuthsMap(),
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
	// 订阅设备注册主题
	d.mqttConn.Sub(1, "/+/+/register")
	log.Println("已订阅设备主题 /+/+/uplink 和注册主题 /+/+/register")

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
			
			// 解析主题获取产品标识和设备编号
			productKey, deviceNo := parseTopic(topic)
			deviceUserName := fmt.Sprintf("%s:%s", productKey, deviceNo)
			
			log.Printf("解析主题: %s -> 产品: %s, 设备: %s", topic, productKey, deviceNo)

			// 认证设备
			isAuth := d.authMap.Has(deviceUserName)
			msg.Device.ProductKey = productKey
			msg.Device.DeviceNo = deviceNo
			
			// 检查是否为注册消息
			if strings.HasSuffix(topic, "/register") {
				log.Printf("收到设备注册消息: %s", deviceUserName)
				d.AuthDevice(msg)
				continue
			}

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

// AuthDevice 设备认证
func (d *DeviceChanMqtt) AuthDevice(msg data.KvMsg) (err error) {
	productKey := fmt.Sprintf("%v", msg.Device.ProductKey)
	deviceNo := msg.Device.DeviceNo
	deviceUserName := fmt.Sprintf("%s:%s", productKey, deviceNo)
	
	// 这里应该调用设备管理服务的API进行认证
	// 验证产品和设备是否在设备中心存在
	log.Printf("设备认证: %s", deviceUserName)
	
	// 添加到认证列表
	d.authMap.Add(deviceUserName)
	log.Printf("设备认证成功，已添加到认证列表: %s", deviceUserName)
	
	// 触发设备上线状态更新
	d.StateDevice(msg)
	return nil
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

// parseTopic 解析MQTT主题，提取产品标识和设备编号
// 主题格式: /产品标识/设备编号/操作类型
func parseTopic(topic string) (productKey, deviceNo string) {
	// 分割主题
	parts := strings.Split(topic, "/")
	// 确保主题格式正确
	if len(parts) >= 3 {
		productKey = parts[1]  // 第二个部分是产品标识
		deviceNo = parts[2]    // 第三个部分是设备编号
	}
	return
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
