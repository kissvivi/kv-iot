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
	return &Influx{client: influxdb2.NewClient(config.CONFIG.Datasource.Influx.Url, config.CONFIG.Datasource.Influx.Token)}
}

// AddFluxData 数据存储格式一个产品一张表
// 存储格式 key:msg.ProductKey:msg.Device.DeviceNo:msg.Property.Identifier
func (i Influx) AddFluxData(msg data.KvMsg) {
	//log.Println("AddFluxData：", msg)
	now := time.Now()
	writeAPI := i.client.WriteAPI(config.CONFIG.Datasource.Influx.Org, config.CONFIG.Datasource.Influx.Bucket)
	field := fmt.Sprintf("%s:%s:%s", msg.ProductKey, msg.Device.DeviceNo, msg.Property.Identifier)
	p := influxdb2.NewPointWithMeasurement(msg.ProductKey.(string)).
		AddTag("value", msg.Property.Identifier).
		AddField(field, msg.Property.Value).SetTime(now)
	writeAPI.WritePoint(p)
	writeAPI.Flush()
	log.Printf(fmt.Sprintf("[%v][AddFluxData]->[field]->%s,[value]->%v", now, field, msg.Property.Value))
	log.Println()
	defer i.client.Close()

}
