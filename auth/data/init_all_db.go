package data

import (
	"fmt"
	"kv-iot/config"
	"kv-iot/db"
)

func InitDB(cfg *config.Config) {
	//初始化数据库
	fmt.Println("正在初始化认证服务数据库连接...")
	baseDB := db.NewBaseDB("mysql")
	// 不打印完整配置信息，避免敏感数据泄露
	baseDB.InitDB(cfg)
	fmt.Println("正在自动迁移数据库表结构...")
	db.MYSQLDB.AutoMigrate(&User{}, &Module{}, &Role{}, &UserRole{}, &Permission{})
	fmt.Println("认证服务数据库初始化完成")
}
