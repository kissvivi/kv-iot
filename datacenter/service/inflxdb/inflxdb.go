package inflxdb

import (
	"context"
	"fmt"
	"kv-iot/config"
	"kv-iot/datacenter/data"
	"log"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// Influx 表示InfluxDB服务

type Influx struct {
	client influxdb2.Client
	org    string
	bucket string
}

// NewInflux 创建一个新的InfluxDB服务实例
func NewInflux() *Influx {
	// 从全局配置获取InfluxDB连接信息
	cfg := config.CONFIG
	influxConfig := cfg.Datasource.Influx

	// 创建客户端
	client := influxdb2.NewClient(influxConfig.Url, influxConfig.Token)
	
	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	health, err := client.Health(ctx)
	if err != nil {
		log.Printf("InfluxDB健康检查失败: %v", err)
	} else if health.Status != "pass" {
		log.Printf("InfluxDB状态异常: %s", health.Status)
	} else {
		log.Println("InfluxDB连接成功")
	}

	return &Influx{
		client: client,
		org:    influxConfig.Org,
		bucket: influxConfig.Bucket,
	}
}

// AddFluxData 添加数据到InfluxDB
func (i *Influx) AddFluxData(msg data.KvMsg) error {
	if msg.Device.DeviceNo == "" {
		return fmt.Errorf("设备编号为空，无法存储数据")
	}

	// 创建写入API
	writeAPI := i.client.WriteAPI(i.org, i.bucket)
	
	// 处理写入错误
	errorsCh := writeAPI.Errors()
	go func() {
		for err := range errorsCh {
			log.Printf("InfluxDB写入错误: %v", err)
		}
	}()

	// 准备数据点
	wp := influxdb2.NewPointWithMeasurement("device_data").
		AddTag("device_no", msg.Device.DeviceNo).
		AddTag("product_key", fmt.Sprintf("%v", msg.Device.ProductKey)).
		AddTag("channel_id", msg.ChannelID).
		SetTime(time.Now())

	// 添加属性值（如果属性标识符不为空）
	if msg.Property.Identifier != "" {
		// 将属性值作为字段添加到数据点
		wp.AddField(msg.Property.Identifier, msg.Property.Value)
	}

	// 写入数据
	writeAPI.WritePoint(wp)
	writeAPI.Flush()

	log.Printf("成功写入InfluxDB: 设备=%s, 测量=%s", 
		msg.Device.DeviceNo, "device_data")
	return nil
}

// Close 关闭InfluxDB连接
func (i *Influx) Close() error {
	if i.client != nil {
		i.client.Close()
		log.Println("InfluxDB连接已关闭")
	}
	return nil
}

// Query 查询InfluxDB数据
func (i *Influx) Query(query string) ([]map[string]interface{}, error) {
	// 创建查询API
	queryAPI := i.client.QueryAPI(i.org)

	// 执行查询
	result, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}

	// 处理查询结果
	var results []map[string]interface{}
	for result.Next() {
		// 直接使用Record获取数据，避免使用Table方法
		row := make(map[string]interface{})
		for k, v := range result.Record().Values() {
			row[k] = v
		}
		results = append(results, row)
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("查询处理失败: %w", result.Err())
	}

	return results, nil
}