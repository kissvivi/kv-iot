package data

type KvEvent struct {
	ID         int64  `json:"id" gorm:"column:id"`
	ProductID  int    `json:"product_id" gorm:"column:product_id"`
	ProductKey string `json:"product_key" gorm:"column:product_key"` // 产品标识
	Name       string `json:"name" gorm:"column:name"`               // 动作名称
	Identifier string `json:"identifier" gorm:"column:identifier"`   // 动作标识符
}

func (m *KvEvent) TableName() string {
	return "kv_event"
}
