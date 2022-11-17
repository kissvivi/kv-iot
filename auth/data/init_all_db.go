package data

import (
	"fmt"
	"github.kissvivi.kv-iot/config"
	"github.kissvivi.kv-iot/db"
)

func InitDB(cfg *config.Config) {
	//初始化数据库
	baseDB := db.NewBaseDB("mysql")
	fmt.Println(cfg)
	baseDB.InitDB(cfg)
	db.MYSQLDB.AutoMigrate(&User{})
}
