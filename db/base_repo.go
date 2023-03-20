package db

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"log"
	"reflect"
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
	FindBy(col T, value T) (err error, result []T)
}
type BaseRepo[T any] struct {
}

// Add
//
//	@Description: 保存操作
//	@receiver baseRepo[T]
//	@param t 实体类
//	@return err
func (br BaseRepo[T]) Add(t T) (err error) {
	log.Println(t)
	err = MYSQLDB.Create(&t).Error
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// Update
//
//	@Description: 更新操作
//	@receiver baseRepo[T]
//	@param t 实体类
//	@return err
func (br BaseRepo[T]) Update(t T) (err error) {
	err = MYSQLDB.Updates(&t).Error
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// Delete
//
//	@Description: 删除操作
//	@receiver baseRepo[T]
//	@param t 实体类
//	@return err
func (br BaseRepo[T]) Delete(t T) (err error) {
	err, sqlText, sqlValues := br.buildWhere(t)
	err = MYSQLDB.Where(sqlText, sqlValues...).Delete(&t).Error
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// FindOneByID
//
//	@Description: 根据id查询
//	@receiver baseRepo[T]
//	@param id
//	@return err
//	@return result 查询结果/结构体
func (br BaseRepo[T]) FindOneByID(id int) (err error, result T) {
	err = MYSQLDB.Where("id", id).First(&result).Error
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// FindAll
//
//	@Description: 查询所有数据
//	@receiver baseRepo[T]
//	@return err
//	@return []result 返回实体列表
func (br BaseRepo[T]) FindAll() (err error, result []T) {
	result = make([]T, 0)
	err = MYSQLDB.Find(&result).Error
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// FindBy
//
//	@Description: 根据某个字段查询
//	@receiver baseRepo[T]
//	@param m 字段键值对
//	@return err
//	@return []result 返回实体列表
func (br BaseRepo[T]) FindBy(m map[string]interface{}) (err error, result []T) {
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
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// FindByStruct
//
//	@Description: 根据某个结构体查询/只支持 单层 结构体
//	@receiver baseRepo[T]
//	@param st 结构体
//	@return err
//	@return []result 返回实体列表
func (br BaseRepo[T]) FindByStruct(st any) (err error, result []T) {
	err, sqlText, sqlValues := br.buildWhere(st)
	if err != nil {
		return err, nil
	}
	err = MYSQLDB.Where(sqlText, sqlValues...).Find(&result).Error
	if err != nil {
		log.Println(err)
		return
	}
	return
}

func (br BaseRepo[T]) buildWhere(st any) (err error, sqlText string, sqlValues []interface{}) {

	var m = make(map[string]interface{})
	var t = reflect.TypeOf(st)
	var v = reflect.ValueOf(st)

	if st == nil {
		return errors.New("struct is not null"), "", nil
	}
	for i := 0; i < t.NumField(); i++ {

		field := t.Field(i) //每个结构体对象
		log.Println(v.Field(i).Interface())

		//TODO 实际值为零值怎么办
		switch vType := v.Field(i).Interface().(type) {
		case int, int8, int32, int64:
			value := v.Field(i).Int()
			if value != 0 {
				log.Printf("%T,int:%+v\n", value, value)
				m[field.Tag.Get("json")] = value
			}
		case uint, uint8, uint16, uint32, uint64:
			value := v.Field(i).Int()
			if value != 0 {
				log.Printf("%T,uint:%+v\n", value, value)
				m[field.Tag.Get("json")] = value
			}
		case float64, float32:
			value := v.Field(i).Float()
			if value != 0 {
				m[field.Tag.Get("json")] = value
			}
		case string:
			value := v.Field(i).String()
			if value != "" {
				m[field.Tag.Get("json")] = value
			}
		case nil:
			log.Printf("为空类型：%v\n", vType)
		case gorm.Model:
			value := v.Field(i).Interface()
			log.Printf("为gorm.Model结构体类型：%v\n", vType)
			id := value.(gorm.Model).ID
			deletedAt := value.(gorm.Model).DeletedAt
			updatedAt := value.(gorm.Model).UpdatedAt
			createdAt := value.(gorm.Model).CreatedAt

			if id != 0 {
				m["id"] = id
			}
			if !deletedAt.Time.IsZero() {
				log.Printf("删除时间：%v\n", value)
				m["deleted_at"] = deletedAt
			}
			if !updatedAt.IsZero() {
				log.Printf("更新时间：%v\n", value)
				m["updated_at"] = updatedAt
			}
			if !createdAt.IsZero() {
				log.Printf("创建时间：%v\n", value)
				m["created_at"] = createdAt
			}
		default:
			//isNil := reflect.ValueOf(value).IsNil()
			//if isNil {
			//	m[field.Tag.Get("json")] = value
			//}
		}

	}
	i := 0
	for name, value := range m {
		sqlText += fmt.Sprintf("%s = ?", name)
		sqlValues = append(sqlValues, value)
		if i < len(m)-1 {
			//fmt.Println(fmt.Sprintf("%d_%d", len(m), i))
			sqlText += " and "
		}
		i++
	}

	return
}
