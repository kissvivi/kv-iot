package db

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"kv-iot/config"
)

type MysqlDB struct {
}

//func (m MysqlDB) SetConfig(conf *config.Config) {
//	m.Url = conf.Datasource.Mysql.Url
//	m.UserName = conf.Datasource.Mysql.Username
//	m.Password = conf.Datasource.Mysql.Password
//	m.DBName = conf.Datasource.Mysql.Dbname
//
//	fmt.Println(m.Url)
//}

var MYSQLDB *gorm.DB

func (m MysqlDB) InitDB(conf *config.Config) {

	fmt.Println(m)
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True&loc=Local",
			conf.Datasource.Mysql.Username, conf.Datasource.Mysql.Password, conf.Datasource.Mysql.Url, conf.Datasource.Mysql.Dbname), // DSN data source name
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据当前 MySQL 版本自动配置
	}), &gorm.Config{})
	if err != nil {
		panic(any(err))
	}
	MYSQLDB = db
}

var _ BaseDB = (*MysqlDB)(nil)
