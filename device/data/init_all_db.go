package data

import (
	"fmt"
	"kv-iot/config"
	"kv-iot/db"
	"log"
)

func InitDB(cfg *config.Config) {
	//初始化数据库
	baseDB := db.NewBaseDB("mysql")
	fmt.Println("正在初始化设备管理服务数据库连接...")
	baseDB.InitDB(cfg)
	fmt.Println("正在自动迁移数据库表结构...")
	db.MYSQLDB.AutoMigrate(
		Channels{},
		Products{},
		Devices{},
		KvAction{},
		KvEvent{},
		KvProperty{},
		ChannelInstance{},
		DeviceConn{},
		ProductProtocolProfile{},
		ProductPointBinding{},
		DevicePointBinding{},
	)
	ensureAutoIncrement("kv_action")
	ensureAutoIncrement("kv_event")
}

func ensureAutoIncrement(table string) {
	if db.MYSQLDB == nil || !db.MYSQLDB.Migrator().HasTable(table) {
		return
	}
	query := fmt.Sprintf("ALTER TABLE `%s` MODIFY `id` bigint unsigned NOT NULL AUTO_INCREMENT", table)
	if err := db.MYSQLDB.Exec(query).Error; err != nil {
		log.Printf("规范化 %s.id 自增失败: %v", table, err)
	}
}
