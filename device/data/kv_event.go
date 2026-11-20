package data

import "gorm.io/gorm"

type KvEvent struct {
	//ID         int64  `json:"id" gorm:"column:id"`
	gorm.Model
	ProductID  int64  `json:"product_id" gorm:"column:product_id"`   // 修改为int64以保持一致
	ProductKey string `json:"product_key" gorm:"column:product_key"` // 产品标识
	Name       string `json:"name" gorm:"column:name"`               // 事件名称
	Identifier string `json:"identifier" gorm:"column:identifier"`   // 事件标识符
	Severity   string `json:"severity" gorm:"column:severity;size:16;default:info"`
	DataSchema string `json:"data_schema" gorm:"column:data_schema;type:text"`
}

func (m *KvEvent) TableName() string {
	return "kv_event"
}
