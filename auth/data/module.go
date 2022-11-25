package data

import "gorm.io/gorm"

type Module struct {
	gorm.Model
	Name       string `json:"name" gorm:"column:name"`               //资源名
	Uri        string `json:"uri" gorm:"column:uri"`                 //资源地址
	Code       string `json:"code" gorm:"column:code"`               //权限code
	ParentID   int64  `json:"parent_id" gorm:"column:parent_id"`     //父亲资源id
	ParentCode int64  `json:"parent_code" gorm:"column:parent_code"` //父亲权限code
	Enable     int    `json:"enable" gorm:"column:enable"`           //是否有效
}

func (m *Module) TableName() string {
	return "module"
}
