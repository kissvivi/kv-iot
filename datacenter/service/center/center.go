package center

import (
	"encoding/json"
	"kv-iot/datacenter/data"
	"kv-iot/datacenter/service/center/inflxdb"
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
	decodeScript := `
		// BinaryToJSON function from the JavaScript decoder
		// You can modify this script to match your specific parsing logic
		// The script should populate the decodedMsg object with the parsed data
		
		var decodedMsg = {};

		// Example: Accessing the xxx variable from Go
		decodedMsg.xxx = xxx;

		// Example: Parsing properties
		decodedMsg.property = {
			name: "Temperature",
			identifier: "temperature",
			dataType: "float",
			unit: "°C",
			is_rw: "r",
			sub_property: 0,
			sub_property_id: "",
			value: 25.5
		};

		// Example: Parsing actions
		decodedMsg.action = {
			name: "TurnOn",
			identifier: "turn_on"
		};

		// Example: Parsing events
		decodedMsg.event = {
			name: "Error",
			identifier: "error"
		};

		// Example: Parsing device
		decodedMsg.device = {
			product_key: "your_product_key",
			name: "Device 1",
			device_no: "device-001"
		};

		decodedMsg;  // Return the decoded message object
	`

	// Convert binary data to JSON using the JavaScript decoder
	b, err := pkg.BinaryToJSON(0, nil, decodeScript, c.Data)
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshal JSON into KvMsg struct
	err = json.Unmarshal(b, &msg)
	if err != nil {
		log.Fatal(err)
	}

	msg.ChannelID = c.ChannelID

	return msg
}

// Encode 编码
func (c *Center) Encode(msg data.KvMsg) interface{} {
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

func (c *Center) ToInflux(msg data.KvMsg) {
	inflxdb.NewInflux().AddFluxData(msg)
}
