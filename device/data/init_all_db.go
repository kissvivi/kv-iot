package data

import (
	"fmt"
	"kv-iot/config"
	"kv-iot/db"
)

func InitDB(cfg *config.Config) {
	//初始化数据库
	baseDB := db.NewBaseDB("mysql")
	fmt.Println("正在初始化设备管理服务数据库连接...")
	baseDB.InitDB(cfg)
	fmt.Println("正在自动迁移数据库表结构...")
	db.MYSQLDB.AutoMigrate(Channels{}, Products{}, Devices{}, KvAction{}, KvEvent{}, KvProperty{})
}
