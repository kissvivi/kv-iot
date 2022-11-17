package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"kv-iot/auth/api"
	"kv-iot/auth/data"
	v1 "kv-iot/auth/endpoint/http/v1"
	"kv-iot/config"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

// serverCmd represents the endpoint command
var serverCmd = &cobra.Command{
	Use:   "endpoint",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: runServer,
}

//type Application struct {
//	name string
//	version string
//	httpServer *http.Server
//}

func runServer(cmd *cobra.Command, args []string) {
	//初始化config
	//初始化application
	//启动数据库
	cfg, err := config.InitConfig()
	if err != nil {
		panic(err)
	}
	//初始化DB
	data.InitDB(cfg)

	engine := v1.InitRouter(api.BaseApi{})
	s := initServer(cfg, engine)
	//go initGrpcServer()
	fmt.Println(`
	 ___  __    ___      ___             ___  ________  _________   
	|\  \|\  \ |\  \    /  /|           |\  \|\   __  \|\___   ___\ 
	 \ \  \/  /|\ \  \  /  / /___________\ \  \ \  \|\  \|___ \  \_| 
 	  \ \   ___  \ \  \/  / /\____________\ \  \ \  \\\  \   \ \  \  
  	   \ \  \\ \  \ \    / /\|____________|\ \  \ \  \\\  \   \ \  \ 
   	    \ \__\\ \__\ \__/ /                 \ \__\ \_______\   \ \__\
         \|__| \|__|\|__|/                   \|__|\|_______|    \|__|`)

	fmt.Printf(`
	 	欢迎使用  kv-iot
	 	服务版本 : %v 
	 	服务运行地址 : %v
	`, cfg.Application.AuthServer.Version, s.Addr)

	err = s.ListenAndServe()
	if err != nil {
		return
	}

}

func initGrpcServer() {
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}

func initServer(setting *config.Config, r *gin.Engine) *http.Server {
	s := &http.Server{
		Addr:           fmt.Sprintf("%v:%v", setting.Application.AuthServer.HttpServer.Host, setting.Application.AuthServer.HttpServer.Port),
		Handler:        r,
		ReadTimeout:    5 * time.Minute,
		WriteTimeout:   5 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}
	return s
}

func Execute() {

	if err := serverCmd.Execute(); err != nil {
		//logrus.Error(err)
		fmt.Errorf("%s", err)
		os.Exit(1)
	}
}

func main() {
	Execute()

	//println(int(1 << uint(13)) & 12)
}
