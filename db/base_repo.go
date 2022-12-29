package db

import (
	"fmt"
	"log"
)

// BaseRepoI
//
//	BaseRepoI[T any]
//	@Description: 数据库操作基础接口
type BaseRepoI[T any] interface {
	Add(t T) (err error)
	Update(t T) (err error)
	Delete(t T) (err error)
	FindOneByID(id int) (err error, result T)
	FindAll() (err error, result []T)
	FindOneBy(col T, value T) (err error, result []T)
}

type baseRepo[T any] struct{}

// Add
//
//	@Description: 保存操作
//	@receiver baseRepo[T]
//	@param t 实体类
//	@return err
func (baseRepo[T]) Add(t T) (err error) {
	log.Println(t)
	err = MYSQLDB.Create(&t).Error
	if err != nil {
		log.Println(err)
	}
	return
}

// Update
//
//	@Description: 更新操作
//	@receiver baseRepo[T]
//	@param t 实体类
//	@return err
func (baseRepo[T]) Update(t T) (err error) {
	return MYSQLDB.Updates(&t).Error
}

// Delete
//
//	@Description: 删除操作
//	@receiver baseRepo[T]
//	@param t 实体类
//	@return err
func (baseRepo[T]) Delete(t T) (err error) {
	return MYSQLDB.Delete(&t).Error
}

// FindOneByID
//
//	@Description: 根据id查询
//	@receiver baseRepo[T]
//	@param id
//	@return err
//	@return result 查询结果/结构体
func (baseRepo[T]) FindOneByID(id int) (err error, result T) {
	err = MYSQLDB.Where("id", id).First(&result).Error
	return
}

// FindAll
//
//	@Description: 查询所有数据
//	@receiver baseRepo[T]
//	@return err
//	@return []result 返回实体列表
func (baseRepo[T]) FindAll() (err error, result []T) {
	result = make([]T, 0)
	err = MYSQLDB.Find(&result).Error
	return
}

// FindBy
//
//	@Description: 根据某个字段查询
//	@receiver baseRepo[T]
//	@param m 字段键值对
//	@return err
//	@return []result 返回实体列表
func (baseRepo[T]) FindBy(m map[string]interface{}) (err error, result []T) {
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
	err = MYSQLDB.Where(sql, sqlValues...).Find(&result).Error
	return
}