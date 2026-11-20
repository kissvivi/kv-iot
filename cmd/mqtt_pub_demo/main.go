package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"kv-iot/pkg/mqtt"
)

// DeviceData 定义设备数据结构
type DeviceData struct {
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	Timestamp   int64   `json:"timestamp"`
}

func main() {
	// 配置MQTT客户端
	mqttConfig := mqtt.NewConfigMqtt(
		"127.0.0.1",     // MQTT服务器地址
		1883,            // MQTT端口（int类型）
		"mqtt_pub_demo", // 客户端ID
		"admin",         // 用户名
		"password",      // 密码
	)

	// 创建MQTT客户端
	client := mqtt.NewMqtt(mqttConfig)
	if client == nil {
		log.Fatalf("创建MQTT客户端失败")
	}
	defer client.Conn.Disconnect(250)

	// 等待连接建立
	time.Sleep(2 * time.Second)

	// 设备信息
	productKey := "TMP"   // 产品标识
	deviceNumber := "001" // 设备编号

	// 上行消息主题 (设备到服务器)
	uplinkTopic := fmt.Sprintf("/%s/%s/uplink", productKey, deviceNumber)

	// 下行消息主题 (服务器到设备)
	downlinkTopic := fmt.Sprintf("/%s/%s/downlink", productKey, deviceNumber)

	// 创建测试数据
	deviceData := DeviceData{
		Temperature: 25.5,
		Humidity:    60.2,
		Timestamp:   time.Now().Unix(),
	}

	// 序列化数据为JSON
	dataJSON, err := json.Marshal(deviceData)
	if err != nil {
		log.Fatalf("序列化数据失败: %v", err)
	}

	// 发布上行消息示例
	log.Printf("开始发布上行消息到 %s: %s", uplinkTopic, string(dataJSON))
	client.Pub(1, uplinkTopic, string(dataJSON))
	log.Println("上行消息发布完成")

	// 创建控制指令数据
	controlCmd := map[string]interface{}{
		"action": "set_threshold",
		"params": map[string]interface{}{
			"temp_high": 30.0,
			"temp_low":  10.0,
		},
		"timestamp": time.Now().Unix(),
	}

	// 序列化控制指令为JSON
	cmdJSON, err := json.Marshal(controlCmd)
	if err != nil {
		log.Fatalf("序列化控制指令失败: %v", err)
	}

	// 发布下行控制指令
	log.Printf("开始发布下行控制指令到 %s: %s", downlinkTopic, string(cmdJSON))
	client.Pub(1, downlinkTopic, string(cmdJSON))
	log.Println("下行控制指令发布完成")

	// 等待消息发送完成
	time.Sleep(2 * time.Second)
	log.Println("MQTT消息发送完成")
}
