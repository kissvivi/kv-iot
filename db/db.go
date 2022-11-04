package db

import "github.kissvivi.kv-iot/config"

type BaseDB interface {
	InitDB()                         //初始化链接驱动
	AutoMigrates(dst ...interface{}) //初始化表
	SetConfig(conf *config.Config)
}

func NewBaseDB(t string) BaseDB {
	switch t {
	case "mysql":
		return &MysqlDB{}
	//case "oracle":
	//	return &oracleDB{}
	default:
		return &MysqlDB{}
	}
}

func main() {
	cfg, err := config.InitConfig()
	if err != nil {
		panic(err)
	}
	baseDB := NewBaseDB("mysql")
	baseDB.SetConfig(cfg)
	baseDB.InitDB()
	baseDB.AutoMigrates()
}
