package data

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"kv-iot/db"
)

var (
	ErrDBSelect = errors.New("数据库查询异常")
	ErrDBAdd    = errors.New("数据库添加异常")
	ErrDBUpdate = errors.New("数据库更新异常")
	ErrDBDelete = errors.New("数据库删除异常")
)

//type AuthRepoI [T any] interface {
//	Add(t T) error
//	Update(t T) error
//	Delete(t T) error
//	Select(t T) error
//}

type AuthRepo[T any] struct {
	db *gorm.DB
}

func NewAuthRepo() *AuthRepo[any] {
	return &AuthRepo[any]{db: db.MYSQLDB}
}

func (a AuthRepo[T]) Add(t T) (err error) {
	fmt.Println(t)
	err = a.db.Create(&t).Error
	if err != nil {
		fmt.Println(err)
	}
	return
}

func (a AuthRepo[T]) Update(t T) error {
	return a.db.Updates(&t).Error
}

func (a AuthRepo[T]) Delete(t T) error {
	return a.db.Delete(&t).Error
}

func (a AuthRepo[T]) Select(t T) (err error, result T) {
	err = a.db.Model(&result).First(&t).Error
	return
}

func (a AuthRepo[T]) FindAll(t T) (err error, result []T) {
	err = a.db.Model(&result).Find(&t).Error
	return
}
