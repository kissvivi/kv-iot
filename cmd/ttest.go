package main

import (
	"database/sql"
	"fmt"
	"reflect"
)

func Bit(x int) int {
	return int(1 << uint(x))
}

type roles struct {
	roleId   int
	roleName string
}
type User struct {
	Name     string
	Age      bool
	Email    string
	NickName string
	Telphone int
	Roles    roles
}

//func main() {
//	u := User{Name: "Name", Age: true, Email: "xxxx@afanty3d.com", NickName: "omni360", Telphone: 1, Roles: roles{roleId: 1001, roleName: "administrator"}}
//	fmt.Println(u)
//	Info(u)
//
//}

func Info(o interface{}) {
	t := reflect.TypeOf(o)
	fmt.Println("Type:", t.Name())

	v := reflect.ValueOf(o)
	fmt.Println("Fileds:")
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		val := v.Field(i).Interface()
		if f.Type.Kind() == 1 {
			switch val {
			case true:
				val = 1
			case false:
				val = 0
			default:
				val = 0
			}
		}
		fmt.Printf("%6s : %v %v\n", f.Name, f.Type, val)
	}
}

//func main() {
//	//a := [5]int{1, 2, 3, 4, 5}
//	//
//	//for i, value := range a {
//	//
//	//	i = 5
//	//	fmt.Println(i, value)
//	//}
//	//
//	//for i := 0; i < len(a); i++ {
//	//	i = 4
//	//	fmt.Println(i, a[i])
//	//}
//
//	fmt.Println(Bit(9))
//}

func DoQuery(db *sql.DB, sqlInfo string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Query(sqlInfo, args...)
	if err != nil {
		return nil, err
	}
	columns, _ := rows.Columns()
	columnLength := len(columns)
	cache := make([]interface{}, columnLength) //临时存储每行数据
	for index, _ := range cache {              //为每一列初始化一个指针
		var a interface{}
		cache[index] = &a
	}
	var list []map[string]interface{} //返回的切片
	for rows.Next() {
		_ = rows.Scan(cache...)

		item := make(map[string]interface{})
		for i, data := range cache {
			v := *data.(*interface{})
			fmt.Println(fmt.Sprintf("外：key:%v,value:%v", columns[i], v))
			switch v.(type) {
			case []byte:
				v = string(v.([]byte))
				fmt.Println(fmt.Sprintf("[]byte：key:%v,value:%v", columns[i], v))
				//取实际类型
			case nil:
				v = ""
			case string:
				fmt.Println("string", v)
			}
			item[columns[i]] = v
		}
		list = append(list, item)
	}
	_ = rows.Close()
	return list, nil
}

func main() {
	//cfg, err := config.InitConfig()
	//if err != nil {
	//	panic(any(err))
	//}
	////初始化DB
	//data.InitDB(cfg)
	//db, _ := db.MYSQLDB.DB()
	//fmt.Println(DoQuery(db, "select * from user"))

	//s := "abcd你好"
	//fmt.Println(s)
	//for _, ss := range s {
	//	fmt.Println(ss)
	//}

	fmt.Println(1111)

}
