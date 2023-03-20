package data

import (
	"fmt"
	"kv-iot/config"
	"kv-iot/db"
)

func InitDB(cfg *config.Config) {
	//初始化数据库
	baseDB := db.NewBaseDB("mysql")
	fmt.Println(cfg)
	baseDB.InitDB(cfg)
	db.MYSQLDB.AutoMigrate(Channels{}, Products{}, Devices{}, KvAction{}, KvEvent{}, KvProperty{}, KvProperty{})
}
