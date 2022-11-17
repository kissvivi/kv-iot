package db

import "kv-iot/config"

type BaseDB interface {
	InitDB(conf *config.Config) //初始化链接驱动
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
	baseDB.InitDB(cfg)
}
