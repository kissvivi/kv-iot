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

// serverCmd represents the authentication server command
var serverCmd = &cobra.Command{
	Use:   "run",
	Short: "启动认证服务",
	Long:  `启动kv-iot平台的认证服务，负责用户认证、角色管理和权限控制等核心功能。`,
	Run:   runServer,
}

func runServer(cmd *cobra.Command, args []string) {
	//初始化配置
	//初始化应用
	//启动数据库
	log.Println("正在初始化认证服务...")
	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatalf("配置初始化失败: %v", err)
	}
	log.Println("配置初始化成功")
	//初始化DB
	data.InitDB(cfg)

	//new 建立依赖
	baseApi := api.NewBaseApi(api.NewUserApiImpl(service.NewBaseService(service.NewUserServiceImpl(repo.UserRepo{}), service.NewRoleServiceImpl(repo.RoleRepo{}))))

	engine := v1.InitRouter(baseApi)
	s := initServer(cfg, engine)
	//启动gRPC服务
	go initGrpcServer(cfg)
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
	fmt.Println()

	// 输出服务基本配置信息，不包含敏感数据
	log.Println("===== 认证服务配置信息 =====")
	log.Printf("应用名称: kv-iot")
	log.Printf("服务版本: %v", cfg.Application.AuthServer.Version)
	log.Printf("HTTP服务地址: %v:%v", cfg.Application.AuthServer.HttpServer.Host, cfg.Application.AuthServer.HttpServer.Port)
	log.Printf("gRPC服务地址: %v:%v", cfg.Application.AuthServer.GrpcServer.Host, cfg.Application.AuthServer.GrpcServer.Port)
	log.Printf("数据库: [已隐藏敏感信息]")
	log.Printf("数据库名称: %v", cfg.Datasource.Mysql.Dbname)
	log.Println("==========================")

	err = s.ListenAndServe()
	if err != nil {
		log.Printf("认证服务启动失败: %v", err)
		return
	}

}

func initGrpcServer(cfg *config.Config) {
	// 使用配置中的gRPC端口，默认为9001
	grpcAddr := fmt.Sprintf("%v:%v", cfg.Application.AuthServer.GrpcServer.Host, cfg.Application.AuthServer.GrpcServer.Port)
	log.Printf("正在启动gRPC服务，监听地址: %s", grpcAddr)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Printf("gRPC监听失败: %v", err)
		return
	}

	grpcServer := grpc.NewServer()
	log.Printf("gRPC服务启动成功，监听地址: %s", grpcAddr)

	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("gRPC服务停止: %v", err)
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
	// 先添加子命令
	serverCmd.AddCommand(versionCmd)

	if err := serverCmd.Execute(); err != nil {
		log.Printf("命令执行失败: %v", err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of cobrademo",
	Long:  `All software has versions. This is cobrademo's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cobrademo version is v1.0")
	},
}
