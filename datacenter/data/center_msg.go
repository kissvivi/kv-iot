package data

// KvMsg 平台统一数据格式->对接到物模型
type KvMsg struct {
	ChannelID  string      `json:"channel_id"`  // 产品通讯通道id
	ProductKey interface{} `json:"product_key"` // 产品标识
	Device     Device      `json:"device"`
	Property   Property    `json:"property"`
	Action     Action      `json:"action"`
	Event      Event       `json:"event"`
}

type Property struct {
	Name          string      `json:"name"`            // 属性名称
	Identifier    string      `json:"identifier"`      // 属性标识符
	DataType      string      `json:"dataType"`        // 属性数据类型
	Unit          string      `json:"unit"`            // 属性单位
	IsRw          string      `json:"is_rw"`           // 是否可读写(r,w,rw)
	SubProperty   int16       `json:"sub_property"`    // 是否有子属性
	SubPropertyID string      `json:"sub_property_id"` // 属性id
	Value         interface{} `json:"value"`           //值
}

type Action struct {
	Name       string `json:"name"`       // 动作名称
	Identifier string `json:"identifier"` // 动作标识符
}

type Event struct {
	Name       string `json:"name"`       // 事件名称
	Identifier string `json:"identifier"` // 事件标识符
}

type Device struct {
	ProductKey interface{} `json:"product_key"`
	Name       string      `json:"name"` // 设备名称
	DeviceNo   string      `json:"device_no"`
}
