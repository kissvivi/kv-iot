package data

import "gorm.io/gorm"

type Channels struct {
	gorm.Model
	//ID         int64  `json:"id" gorm:"column:id"`
	Name       string `json:"name" gorm:"column:name"`               // 通道名称
	Desc       string `json:"desc" gorm:"column:desc"`               // 通道描述
	Encode     string `json:"encode" gorm:"column:encode"`           // 编码脚本
	Decode     string `json:"decode" gorm:"column:decode"`           // 解码脚本
	ScriptType string `json:"script_type" gorm:"column:script_type"` // 编解码脚本类型
}

func (m *Channels) TableName() string {
	return "channels"
}

//type ChannelsRepo interface {
//	Add(channels Channels)
//}
