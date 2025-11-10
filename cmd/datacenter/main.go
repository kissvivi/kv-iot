package main

import (
	"fmt"
	"kv-iot/config"
	"kv-iot/datacenter/service/mqttchan"
	"kv-iot/db"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// 全局配置
var (
	cfg *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "kv-iot",
	Short: "kv-iot 数据中心服务",
	Long: `
	 ___  __    ___      ___             ___  ________  _________   
	|\  \|\  \ |\  \    /  /|           |\  \|\   __  \|\___   ___\ 
	 \ \  \/  /|\ \  \  /  / /___________\ \  \ \  \|\  \|___ \  \_| 
   \ \   ___  \ \  \/  / /\____________\ \  \ \  \\\  \   \ \  \  
    \ \  \\ \  \ \    / /\|____________|\ \  \ \  \\\  \   \ \  \ 
     \ \__\\ \__\ \__/ /                 \ \__\ \_______\   \ \__\
      \|__| \|__|\|__|/                   \|__|\|_______|    \|__|

    数据采集中心 - 连接设备与平台的桥梁，统一数据格式处理
`,
}

var serverCmd = &cobra.Command{
	Use:   "run",
	Short: "运行数据中心服务，监听数据并转发数据",
	Long:  `-mqtt 表示使用mqtt监听`,
	Run:   runServer,
}

func init() {
	// 添加命令行参数
	rootCmd.AddCommand(serverCmd)
}

func runServer(cmd *cobra.Command, args []string) {
	// 显示程序说明信息
	fmt.Println(`
	 ___  __    ___      ___             ___  ________  _________   
	|\  \|\  \ |\  \    /  /|           |\  \|\   __  \|\___   ___\ 
	 \ \  \/  /|\ \  \  /  / /___________\ \  \ \  \|\  \|___ \  \_| 
 	  \ \   ___  \ \  \/  / /\____________\ \  \ \  \\\  \   \ \  \  
  	   \ \  \\ \  \ \    / /\|____________|\ \  \ \  \\\  \   \ \  \ 
   	    \ \__\\ \__\ \__/ /                 \ \__\ \_______\   \ \__\
         \|__| \|__|\|__|/                   \|__|\|_______|    \|__|
		 
		 数据采集中心 - 连接设备与平台的桥梁，统一数据格式处理`)
	// 1. 初始化配置
	var err error
	cfg, err = config.InitConfig()
	if err != nil {
		log.Fatalf("初始化配置失败: %v", err)
	}
	log.Printf("配置初始化成功，版本: %s", "0.0.1")

	// 2. 初始化数据库连接
	initDB()

	// 3. 创建并初始化MQTT通道
	mc := mqttchan.NewDeviceChanMqtt()
	mc.Create()

	// 4. 启动设备注册和数据处理
	go func() {
		mc.RegDevice()
	}()

	log.Println("数据中心服务启动成功")

	// 5. 优雅退出处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭数据中心服务...")
	mc.Close()

	log.Println("数据中心服务已关闭")
}

func initDB() {
	// 初始化MySQL连接
	baseDB := db.NewBaseDB("mysql")
	baseDB.InitDB(cfg)
	log.Println("数据库连接初始化成功")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func main() {
	// 注意：serverCmd已经在init()函数中添加过了，这里不需要重复添加
	Execute()
	//mqttchan.SSub()
}
