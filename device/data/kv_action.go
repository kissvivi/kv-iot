package data

import "gorm.io/gorm"

type KvAction struct {
	//ID         int64  `json:"id" gorm:"column:id"`
	gorm.Model
	Name       string `json:"name" gorm:"column:name"`               // 动作名称
	Identifier string `json:"identifier" gorm:"column:identifier"`   // 动作标识符
	ProductKey string `json:"product_key" gorm:"column:product_key"` // 产品标识
	ProductID  int64  `json:"product_id" gorm:"column:product_id"`   // 产品id
}

func (m *KvAction) TableName() string {
	return "kv_action"
}
