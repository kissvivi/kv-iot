package db

import "gorm.io/gorm"

type BaseDB interface {
	InitDB()
	GetDB() gorm.DB
}
