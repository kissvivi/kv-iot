package inflxdb

import (
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"kv-iot/datacenter/data"
	"log"
	"time"
)

const ApiToken = "WqKMMmXViZ_w8msji4J-vY9asCWkwYQhFX6cK9UlDDCB6JdB6tWLuK7-hTVVRnBRmw0CRPS3b3iBtHKOLPVNMg=="

type Influx struct {
	client influxdb2.Client
}

func NewInflux() *Influx {
	return &Influx{client: influxdb2.NewClient("http://172.19.77.116:8086/", ApiToken)}
}

func (i Influx) AddFluxData(msg data.KvMsg) {
	//log.Println("AddFluxData：", msg)
	now := time.Now()
	writeAPI := i.client.WriteAPI("kv-iot", "kv-iot-bucket")
	p := influxdb2.NewPointWithMeasurement(msg.ProductKey.(string)).
		AddTag("value", msg.Property.Identifier).
		AddField(fmt.Sprintf("%s:%s", msg.ProductKey, msg.Property.Identifier), msg.Property.Value).SetTime(now)
	writeAPI.WritePoint(p)
	writeAPI.Flush()
	log.Printf(fmt.Sprintf("[%v][AddFluxData]->[field]->%s:%s,[value]->%v", now, msg.ProductKey, msg.Property.Identifier, msg.Property.Value))
	log.Println()
	defer i.client.Close()

}
