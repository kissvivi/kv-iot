package data

import "gorm.io/gorm"

type Role struct {
	gorm.Model
	Name   string `json:"name" gorm:"column:name"`     //角色名
	Desc   string `json:"desc" gorm:"column:desc"`     //描述
	Enable int    `json:"enable" gorm:"column:enable"` //是否启用
}

func (r *Role) TableName() string {
	return "role"
}
