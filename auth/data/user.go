package data

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	UserName    string `json:"userName" gorm:"column:user_name"`             //用户名
	Password    string `json:"-" gorm:"column:password"`                     //bcrypt密码哈希，兼容迁移期间的历史明文密码
	IsAdminFlag bool   `json:"isAdmin" gorm:"column:is_admin;default:false"` //是否为管理员
}

func (u User) TableName() string {
	return "user"
}

// IsAdmin 方法检查用户是否为管理员
func (u User) IsAdmin() bool {
	// 基于用户的IsAdminFlag字段判断，而非总是返回true
	return u.IsAdminFlag
}
