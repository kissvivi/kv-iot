package data

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	UserName string `json:"userName" gorm:"column:user_name"` //用户名
	Password string `json:"password" gorm:"column:password"`  //密码
}

func (u User) TableName() string {
	return "user"
}
func (u User) IsAdmin() bool {
	return true
}
