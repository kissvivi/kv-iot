package center

import (
	"encoding/json"
	"io"
	"kv-iot/datacenter/data"
	"kv-iot/datacenter/service/center/inflxdb"
	"log"
	"strings"
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
func (c Center) Decode() (msg data.KvMsg) {
	msgS := string(c.Data)
	//TODO 嵌入js解析器

	dec := json.NewDecoder(strings.NewReader(msgS))
	for {
		if err := dec.Decode(&msg); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		//fmt.Printf("%s: %v\n", msg.ProductKey, msg.Property)
	}
	msg.ChannelID = c.ChannelID
	log.Printf("解析完成%+v\n", msg)
	return
}

// Encode 编码
func (c Center) Encode(msg data.KvMsg) interface{} {
	return nil
}

// ToSql TODO 是否提供给js调用
// 存储数据
func ToSql(msg string) {

}

// ToMq  TODO 是否提供给js调用
// 转发数据
func ToMq(msg string) {

}

func (c Center) ToInflux(msg data.KvMsg) {
	inflxdb.NewInflux().AddFluxData(msg)
}
