package center

import (
	"encoding/json"
	"fmt"
	"kv-iot/datacenter/data"
	"kv-iot/datacenter/service/inflxdb"
	sql_service "kv-iot/datacenter/service/sql"
	"kv-iot/pkg"
	"log"
)

type ICenter interface {
	Decode() (msg data.KvMsg)          //解码
	Encode(msg data.KvMsg) interface{} //编码
}

type Center struct {
	ChannelID string `json:"channel_id"` // 产品通讯通道id
	Data      []byte `json:"data"`
}

func NewCenter(channelID string, data []byte) *Center {
	return &Center{ChannelID: channelID, Data: data}
}

// Decode 解码
//func (c *Center) Decode() (msg data.KvMsg) {
//	msgS := string(c.Data)
//	//TODO 嵌入js解析器
//
//	dec := json.NewDecoder(strings.NewReader(msgS))
//	for {
//		if err := dec.Decode(&msg); err == io.EOF {
//			break
//		} else if err != nil {
//			log.Fatal(err)
//		}
//		//fmt.Printf("%s: %v\n", msg.ProductKey, msg.Property)
//	}
//	msg.ChannelID = c.ChannelID
//	log.Printf("解析完成%+v\n", msg)
//	return
//}

func (c *Center) Decode() (msg data.KvMsg) {
	// 初始化消息结构，设置通道ID
	msg.ChannelID = c.ChannelID
	
	// 尝试直接JSON解析
	if err := json.Unmarshal(c.Data, &msg); err == nil {
		log.Printf("直接JSON解析成功: 产品=%v, 设备=%s", msg.Device.ProductKey, msg.Device.DeviceNo)
		return msg
	}

	// JSON解析失败则使用JS脚本解析
	decodeScript := `
		// 从二进制数据解析设备消息的JavaScript脚本
		function parseBinaryData(data) {
			// 实际项目中应根据设备协议实现解析逻辑
			try {
				var decoded = JSON.parse(data.toString());
				return decoded;
			} catch (e) {
				// 如果不是JSON格式，返回包含原始数据的结构
				return {
					property: {
						name: "raw_data",
						identifier: "raw",
						value: data.toString()
					}
				};
			}
		}
		parseBinaryData(binaryData);
	`

	// 使用二进制转JSON工具 - 将二进制数据转换为字符串格式
	dataStr := string(c.Data)
	b, err := pkg.BinaryToJSON(0, map[string]string{"binaryData": dataStr}, decodeScript, nil)
	if err != nil {
		log.Printf("解析数据失败: %v, 原始数据: %s", err, string(c.Data))
		// 创建默认属性，确保消息可存储
		msg.Property = data.Property{
			Name:       "raw_data",
			Identifier: "raw",
			Value:      string(c.Data),
		}
		return msg
	}

	// 解析JSON结果
	err = json.Unmarshal(b, &msg)
	if err != nil {
		log.Printf("JSON转换失败: %v, 中间结果: %s", err, string(b))
		// 创建默认属性，确保消息可存储
		msg.Property = data.Property{
			Name:       "raw_data",
			Identifier: "raw",
			Value:      string(c.Data),
		}
		return msg
	}

	// 确保通道ID已设置
	msg.ChannelID = c.ChannelID
	log.Printf("成功解析消息: 产品=%v, 设备=%s, 属性=%+v", 
		msg.Device.ProductKey, msg.Device.DeviceNo, msg.Property)
	return msg
}

// Encode 编码
func (c *Center) Encode(msg data.KvMsg) interface{} {
	return nil
}

func (c *Center) ToSql(msg data.KvMsg) error {
	// 验证消息有效性
	if msg.Device.DeviceNo == "" {
		return fmt.Errorf("无效的设备消息: 设备编号为空")
	}

	// 创建SQL服务实例
	sqlService := sql_service.NewSqlService()
	if sqlService == nil {
		return fmt.Errorf("创建SQL服务实例失败")
	}

	// 调用存储方法
	if err := sqlService.Store(msg); err != nil {
		return fmt.Errorf("SQL存储失败: %w", err)
	}

	log.Printf("数据已成功存储到SQL数据库: 设备=%s, 产品=%s", 
		msg.Device.DeviceNo, msg.Device.ProductKey)
	return nil
}

func (c *Center) ToMq(msg data.KvMsg) error {
	// TODO: 实现消息队列发送逻辑
	log.Printf("待实现消息队列发送: %+v", msg)
	return fmt.Errorf("消息队列功能未实现")
}

func (c *Center) ToInflux(msg data.KvMsg) error {
	// 验证消息有效性
	if msg.Device.DeviceNo == "" {
		return fmt.Errorf("无效的设备消息: 设备编号为空")
	}

	// 创建InfluxDB服务实例
	influxService := inflxdb.NewInflux()
	if influxService == nil {
		return fmt.Errorf("创建InfluxDB服务实例失败")
	}

	// 调用存储方法
	if err := influxService.AddFluxData(msg); err != nil {
		return fmt.Errorf("InfluxDB存储失败: %w", err)
	}

	log.Printf("数据已成功存储到InfluxDB: 设备=%s, 产品=%s", 
		msg.Device.DeviceNo, msg.Device.ProductKey)
	return nil
}
