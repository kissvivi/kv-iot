package data

import "gorm.io/gorm"

type Permission struct {
	gorm.Model
	RoleID   int64  `json:"role_id" gorm:"column:role_id"`     //角色ID
	ModuleID int64  `json:"module_id" gorm:"column:module_id"` //资源ID
	Code     string `json:"code" gorm:"column:code"`           //权限Code
}

func (p *Permission) TableName() string {
	return "permission"
}
