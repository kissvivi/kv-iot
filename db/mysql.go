package db

import (
	"fmt"
	"github.kissvivi.kv-iot/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MysqlDB struct {
	DB       *gorm.DB
	Url      string
	UserName string
	Password string
	DBName   string
}

func (m MysqlDB) SetConfig(conf *config.Config) {
	m.Url = conf.Database.Mysql.Url
	m.UserName = conf.Database.Mysql.Username
	m.Password = conf.Database.Mysql.Password
	m.DBName = conf.Database.Mysql.Dbname
}

func (m MysqlDB) AutoMigrates(dst ...interface{}) {
	if err := m.DB.AutoMigrate(dst); err != nil {
		panic(err)
	}
}

func (m MysqlDB) InitDB() {

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True&loc=Local",
			m.UserName, m.Password, m.Url, m.DBName), // DSN data source name
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据当前 MySQL 版本自动配置
	}), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	m.DB = db
}

var _ BaseDB = (*MysqlDB)(nil)
