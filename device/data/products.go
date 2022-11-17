package data

import "gorm.io/gorm"

type Products struct {
	//ID         int64  `json:"id" gorm:"column:id"`
	gorm.Model
	Name       string `json:"name" gorm:"column:name"`               // 产品名称
	Desc       string `json:"desc" gorm:"column:desc"`               // 产品介绍
	ChannelID  string `json:"channel_id" gorm:"column:channel_id"`   // 产品通讯通道id
	ProductKey string `json:"product_key" gorm:"column:product_key"` // 产品标识
}

func (m *Products) TableName() string {
	return "products"
}
