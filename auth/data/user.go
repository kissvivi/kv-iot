package data

import "gorm.io/gorm"

type User struct {
	gorm.Model
	UserName string
	Password string
}

func (u *User) TableName() string {
	return "user"
}
func (u User) IsAdmin() bool {

	return true
}
