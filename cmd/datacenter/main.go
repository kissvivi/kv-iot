package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"kv-iot/datacenter/service/mqttchan"
	"os"
)

// TODO 接受参数传入 可配置 ip 等
var rootCmd = &cobra.Command{
	Use:   "kv-iot",
	Short: "kv-iot",
	Long: `
	 ___  __    ___      ___             ___  ________  _________   
	|\  \|\  \ |\  \    /  /|           |\  \|\   __  \|\___   ___\ 
	 \ \  \/  /|\ \  \  /  / /___________\ \  \ \  \|\  \|___ \  \_| 
 	  \ \   ___  \ \  \/  / /\____________\ \  \ \  \\\  \   \ \  \  
  	   \ \  \\ \  \ \    / /\|____________|\ \  \ \  \\\  \   \ \  \ 
   	    \ \__\\ \__\ \__/ /                 \ \__\ \_______\   \ \__\
         \|__| \|__|\|__|/                   \|__|\|_______|    \|__|`,
	// 如果有相关的 action 要执行，请取消下面这行代码的注释
	// Run: func(cmd *cobra.Command, args []string) { },
}

var serverCmd = &cobra.Command{
	Use:   "run",
	Short: "此命令为运行一个dataCenter服务，监听数据并转发数据",
	Long:  `-mqtt表示使用mqtt监听`,
	Run:   ssub,
}

func ssub(cmd *cobra.Command, args []string) {
	mc := mqttchan.NewDeviceChanMqtt()
	// 获取设备数据 步骤
	// 1.建立数据通道
	// 2.认证设备是否合规
	// 3.注册设备
	// 4.改变设备状态
	// 5.注销设备
	mc.Create()
	mc.AuthDevice()
	mc.RegDevice()
	mc.StateDevice()
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func main() {
	rootCmd.AddCommand(serverCmd)
	Execute()
	//mqttchan.SSub()
}
