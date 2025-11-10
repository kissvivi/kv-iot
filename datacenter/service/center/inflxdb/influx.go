package inflxdb

import (
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"kv-iot/config"
	"kv-iot/datacenter/data"
	"log"
	"time"
)

const ApiToken = "WqKMMmXViZ_w8msji4J-vY9asCWkwYQhFX6cK9UlDDCB6JdB6tWLuK7-hTVVRnBRmw0CRPS3b3iBtHKOLPVNMg=="

type Influx struct {
	client influxdb2.Client
}

func NewInflux() *Influx {
	// 如果配置中没有Token，使用默认Token
	token := config.CONFIG.Datasource.Influx.Token
	if token == "" {
		token = ApiToken
	}
	return &Influx{client: influxdb2.NewClient(config.CONFIG.Datasource.Influx.Url, token)}
}

// Close 关闭InfluxDB客户端连接
func (i *Influx) Close() {
	if i.client != nil {
		i.client.Close()
		log.Println("InfluxDB客户端连接已关闭")
	}
}

// AddFluxData 数据存储格式一个产品一张表
func (i *Influx) AddFluxData(msg data.KvMsg) error {
	// 验证消息有效性
	if msg.Device.DeviceNo == "" {
		return fmt.Errorf("设备编号为空")
	}

	// 处理ProductKey类型转换
	productKey := "unknown"
	if pk, ok := msg.ProductKey.(string); ok {
		productKey = pk
	} else if pk != nil {
		productKey = fmt.Sprintf("%v", msg.ProductKey)
	}

	log.Printf("准备写入InfluxDB: 设备=%s, 产品=%s, 属性=%s, 值类型=%T, 值=%v", 
		msg.Device.DeviceNo, productKey, msg.Property.Identifier, msg.Property.Value, msg.Property.Value)

	now := time.Now()
	writeAPI := i.client.WriteAPI(config.CONFIG.Datasource.Influx.Org, config.CONFIG.Datasource.Influx.Bucket)

	// 创建点并设置标签
	p := influxdb2.NewPointWithMeasurement("device_data")
	p.AddTag("product_key", productKey)
	p.AddTag("device_no", msg.Device.DeviceNo)
	
	if msg.Property.Identifier != "" {
		p.AddTag("property", msg.Property.Identifier)
	}

	// 确保至少添加一个数值字段（InfluxDB的基本要求）
	// 方法1：尝试添加实际值
	hasAddedValue := false
	if msg.Property.Value != nil {
		// 尝试直接添加值作为浮点数（如果可能）
		if num, ok := msg.Property.Value.(float64); ok {
			p.AddField("value_double", num)
			hasAddedValue = true
			log.Printf("添加浮点数字段: %f", num)
		} else if num, ok := msg.Property.Value.(int); ok {
			p.AddField("value_int", num)
			hasAddedValue = true
			log.Printf("添加整数字段: %d", num)
		} else if num, ok := msg.Property.Value.(bool); ok {
			p.AddField("value_bool", num)
			hasAddedValue = true
			log.Printf("添加布尔字段: %v", num)
		} else if str, ok := msg.Property.Value.(string); ok {
			// 对于字符串，同时添加字符串字段和一个计数器字段
			p.AddField("value_string", str)
			p.AddField("has_data", 1.0) // 确保有数值字段
			hasAddedValue = true
			log.Printf("添加字符串字段: %s", str)
		} else {
			// 对于其他类型，转换为字符串并添加一个数值字段
			strValue := fmt.Sprintf("%v", msg.Property.Value)
			p.AddField("value_string", strValue)
			p.AddField("has_data", 1.0) // 确保有数值字段
			hasAddedValue = true
			log.Printf("转换并添加字段: 类型=%T, 值=%s", msg.Property.Value, strValue)
		}
	}

	// 方法2：如果上面没有成功添加字段，强制添加默认字段
	if !hasAddedValue {
		// 确保添加一个数值字段（这是InfluxDB的要求）
		p.AddField("default_value", 1.0)
		p.AddField("message_received", true)
		log.Printf("强制添加默认字段，确保点对象有可序列化的字段")
	}

	// 总是添加一个计数器字段，确保有数值字段
	p.AddField("count", 1.0)
	log.Printf("添加计数器字段 'count': 1.0")

	// 设置时间戳
	p.SetTime(now)

	// 写入数据
	log.Printf("准备写入点对象到InfluxDB")
	writeAPI.WritePoint(p)
	log.Printf("刷新写入缓冲区")
	writeAPI.Flush()

	log.Printf("InfluxDB写入操作完成: 设备=%s, 产品=%s", msg.Device.DeviceNo, productKey)
	return nil
}
