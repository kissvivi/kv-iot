package repo

import (
	"gorm.io/gorm"
)

type ModuleRepo[T any] struct {
	db *gorm.DB
}
