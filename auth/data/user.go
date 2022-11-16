package data

import "gorm.io/gorm"

type User struct {
	gorm.Model
	UserName string
	Password string
}

func (u User) IsAdmin() bool {

	return true
}
