package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Application Application `json:"application"`
	Datasource  Datasource  `json:"datasource"`
}

type Application struct {
	DeviceServer DeviceServer `json:"device_server"`
	AuthServer   AuthServer   `json:"auth_server"`
}

type DeviceServer struct {
	Version    string     `json:"version"`
	HttpServer HttpServer `json:"httpserver"`
}

type AuthServer struct {
	Version      string     `json:"version"`
	HttpServer   HttpServer `json:"httpserver"`
	JwtKey       string     `json:"jwt_key"`
	TokenTimeout int64      `json:"token_timeout"`
}

type HttpServer struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	Mode         string `json:"mode"`
	ReadTimeout  int    `json:"read_timeout"` //单位分钟
	WriteTimeout int    `json:"write_timeout"`
}

type Mysql struct {
	//Host     string `json:"host"`
	//Port     string `json:"port"`
	Url      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Dbname   string `json:"dbname"`
}

type Redis struct {
	//Host     string `json:"host"`
	//Port     string `json:"port"`
	Url      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Dbname   string `json:"dbname"`
}

type Datasource struct {
	Mysql Mysql `json:"mysql"`
	Redis Redis `json:"redis"`

	//Host     string `json:"host"`
	//Port     string `json:"port"`
	//Username string `json:"username"`
	//Password string `json:"password"`
	//Dbname   string `json:"dbname"`
}

//type DBType struct {
//	Url      string `json:"url"`
//	Username string `json:"username"`
//	Password string `json:"password"`
//	Dbname   string `json:"dbname"`
//}

func InitConfig() (*Config, error) {
	v := viper.New()

	//v.AddConfigPath(".")
	//v.SetConfigName("config")
	v.SetConfigFile("config.yml")
	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = v.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}
	CONFIG = cfg
	return &cfg, nil
}

var CONFIG Config
