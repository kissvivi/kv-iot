package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"kv-iot/auth/data"
	"kv-iot/auth/data/repo"
	v1 "kv-iot/auth/endpoint/rest/v1"
	"kv-iot/auth/endpoint/rest/v1/api"
	"kv-iot/auth/service"
	"kv-iot/config"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

// serverCmd represents the endpoint command
var serverCmd = &cobra.Command{
	Use:   "run",
	Short: "kv-iot",
	Long:  `kv-iot`,
	Run:   runServer,
}

func runServer(cmd *cobra.Command, args []string) {
	//初始化config
	//初始化application
	//启动数据库
	cfg, err := config.InitConfig()
	if err != nil {
		panic(any(err))
	}
	//初始化DB
	data.InitDB(cfg)

	//new 建立依赖
	baseApi := api.NewBaseApi(api.NewUserApiImpl(service.NewBaseService(service.NewUserServiceImpl(repo.UserRepo{}), service.NewRoleServiceImpl(repo.RoleRepo{}))))

	engine := v1.InitRouter(baseApi)
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
	serverCmd.AddCommand(versionCmd)
	//println(int(1 << uint(13)) & 12)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of cobrademo",
	Long:  `All software has versions. This is cobrademo's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cobrademo version is v1.0")
	},
}
