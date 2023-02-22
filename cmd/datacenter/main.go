package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"kv-iot/datacenter/service/mq"
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
	Short: "kv-iot",
	Long:  `kv-iot`,
	Run:   ssub,
}

func ssub(cmd *cobra.Command, args []string) {
	mq.SSub()
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
	//mq.SSub()
}
