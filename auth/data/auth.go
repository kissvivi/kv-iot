package data

import (
	"errors"
	"fmt"
	"kv-iot/db"
	"log"
)

var (
	ErrDBSelect = errors.New("数据库查询异常")
	ErrDBAdd    = errors.New("数据库添加异常")
	ErrDBUpdate = errors.New("数据库更新异常")
	ErrDBDelete = errors.New("数据库删除异常")
)

type AuthRepoI[T any] interface {
	Add(t T) (err error)
	Update(t T) (err error)
	Delete(t T) (err error)
	FindOneByID(id int) (err error, result T)
	FindAll() (err error, result []T)
	FindOneBy(col T, value T) (err error, result []T)
}

func (a AuthRepo[T]) Add(t T) (err error) {
	log.Println(t)
	err = db.MYSQLDB.Create(&t).Error
	if err != nil {
		log.Println(err)
	}
	return
}

func (a AuthRepo[T]) Update(t T) (err error) {
	return db.MYSQLDB.Updates(&t).Error
}

func (a AuthRepo[T]) Delete(t T) (err error) {
	return db.MYSQLDB.Delete(&t).Error
}

func (a AuthRepo[T]) FindOneByID(id int) (err error, result T) {
	err = db.MYSQLDB.Where("id", id).First(&result).Error
	return
}

func (a AuthRepo[T]) FindAll() (err error, result []T) {
	result = make([]T, 0)
	err = db.MYSQLDB.Find(&result).Error
	return
}
func (a AuthRepo[T]) FindBy(m map[string]interface{}) (err error, result []T) {

	var (
		sql       string
		sqlValues []interface{}
	)
	i := 0
	for name, value := range m {
		sql += fmt.Sprintf("%s = ?", name)
		sqlValues = append(sqlValues, value)
		if i < len(m)-1 {
			fmt.Println(fmt.Sprintf("%d_%d", len(m), i))
			sql += " and "
		}
		i++
	}

	fmt.Println(sql)
	fmt.Println(sqlValues...)

	err = db.MYSQLDB.Where(sql, sqlValues...).Find(&result).Error
	return
}

type AuthRepo[T any] struct {
}

//func NewAuthRepo() *AuthRepo[any] {
//	return &AuthRepo[any]{db: db.MYSQLDB}
//}
