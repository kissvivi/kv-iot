package config

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	Application Application `json:"application"`
	Datasource  Datasource  `json:"datasource"`
}

type Application struct {
	DeviceServer DeviceServer `json:"deviceServer"`
	AuthServer   AuthServer   `json:"authServer"`
}

type DeviceServer struct {
	Version    string     `json:"version"`
	HttpServer HttpServer `json:"httpserver"`
}

type AuthServer struct {
	Version      string     `json:"version"`
	HttpServer   HttpServer `json:"httpserver"`
	JwtKey       string     `json:"jwtKey"`
	TokenTimeout int64      `json:"tokenTimeout"`
}

type HttpServer struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	Mode         string `json:"mode"`
	ReadTimeout  int    `json:"readTimeout"` //单位分钟
	WriteTimeout int    `json:"writeTimeout"`
}

type Mysql struct {
	//Host     string `json:"host"`
	//Port     string `json:"port"`
	Url      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Dbname   string `json:"dbname"`
}

type Influx struct {
	//Host     string `json:"host"`
	//Port     string `json:"port"`
	Url      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"dbname"`
	Org      string `json:"org"`
	Token    string `json:"token"`
	Bucket   string `json:"bucket"`
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
	Mysql  Mysql  `json:"mysql"`
	Redis  Redis  `json:"redis"`
	Influx Influx `json:"influx"`

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
	//v.SetConfigType("json")
	//v.SetConfigFile("config.json")
	v.SetConfigFile("config.yaml")
	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = v.Unmarshal(&cfg)
	ss, _ := json.Marshal(cfg)
	fmt.Println(string(ss))
	if err != nil {
		return nil, err
	}
	CONFIG = cfg
	return &cfg, nil
}

var CONFIG Config
