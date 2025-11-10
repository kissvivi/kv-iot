package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"kv-iot/config"
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
	v1 "kv-iot/device/endpoint/http/v1"
	"kv-iot/device/endpoint/http/v1/api"
	"kv-iot/device/endpoint/http/v1/api/device"
	"kv-iot/device/endpoint/http/v1/api/product"
	"kv-iot/device/service"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

// serverCmd represents the device management server command
var serverCmd = &cobra.Command{
	Use:   "run",
	Short: "启动设备管理服务",
	Long:  `启动kv-iot平台的设备管理服务，负责设备信息管理、产品管理和通道配置管理等核心功能。`,
	Run:   runServer,
}

//type Application struct {
//	name string
//	version string
//	httpServer *rest.Server
//}

func runServer(cmd *cobra.Command, args []string) {
	// 初始化配置
	// 初始化应用
	// 启动数据库连接
	log.Println("正在初始化设备管理服务...")
	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatalf("配置初始化失败: %v", err)
	}
	log.Println("配置初始化成功")
	//初始化DB
	data.InitDB(cfg)
	//new 建立依赖
	//service实现
	channimpl := service.NewChannelsServiceImpl(repo.ChannelsRepo{})
	deviceImpl := service.NewDevicesServiceImpl(repo.DevicesRepo{})
	kvactionImpl := service.NewKvActionServiceImpl(repo.KvActionRepo{})
	kveventImpl := service.NewKvEventServiceImpl(repo.KvEventRepo{})
	kvpropetyImpl := service.NewKvPropertyServiceImpl(repo.KvPropertyRepo{})
	productsImpl := service.NewProductsServiceImpl(repo.ProductsRepo{})
	//base service
	baseService := service.NewBaseService(channimpl, deviceImpl, kvactionImpl, kveventImpl, kvpropetyImpl, productsImpl)
	baseApi := api.NewBaseApi(device.NewApiDevice(baseService), product.NewApiProduct(baseService))

	engine := v1.InitRouter(baseApi)
	s := initServer(cfg, engine)
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
	`, cfg.Application.DeviceServer.Version, s.Addr)
	// 输出服务基本配置信息，不包含敏感数据
	log.Println("===== 配置信息 =====")
	log.Printf("应用名称: kv-iot")
	log.Printf("服务版本: %v", cfg.Application.DeviceServer.Version)
	log.Printf("HTTP服务地址: %v:%v", cfg.Application.DeviceServer.HttpServer.Host, cfg.Application.DeviceServer.HttpServer.Port)
	log.Printf("gRPC服务地址: %v:%v", cfg.Application.DeviceServer.GrpcServer.Host, cfg.Application.DeviceServer.GrpcServer.Port)
	log.Printf("数据库: %v", maskPassword(cfg.Datasource.Mysql.Url))
	log.Printf("数据库名称: %v", cfg.Datasource.Mysql.Dbname)
	log.Printf("数据库用户: %v", cfg.Datasource.Mysql.Username)
	log.Println("====================")
	err = s.ListenAndServe()
	if err != nil {
		return
	}

}

func initGrpcServer(cfg *config.Config) {
	// 使用配置中的gRPC端口，默认为9000
	grpcAddr := fmt.Sprintf("%v:%v", cfg.Application.DeviceServer.GrpcServer.Host, cfg.Application.DeviceServer.GrpcServer.Port)
	log.Printf("正在初始化gRPC服务，监听地址: %s", grpcAddr)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Printf("gRPC监听失败: %v", err)
		return // 使用return而非Fatalf，避免HTTP服务也受影响
	}

	grpcServer := grpc.NewServer()
	log.Printf("gRPC服务已启动，监听地址: %s", grpcAddr)

	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("gRPC服务异常: %v", err)
	}
}

func initServer(setting *config.Config, r *gin.Engine) *http.Server {
	s := &http.Server{
		Addr:           fmt.Sprintf("%v:%v", setting.Application.DeviceServer.HttpServer.Host, setting.Application.DeviceServer.HttpServer.Port),
		Handler:        r,
		ReadTimeout:    5 * time.Minute,
		WriteTimeout:   5 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}
	return s
}

func Execute() {
	if err := serverCmd.Execute(); err != nil {
		log.Printf("命令执行失败: %v", err)
		os.Exit(1)
	}
}

// maskPassword 隐藏连接字符串中的密码信息
func maskPassword(connStr string) string {
	// 简单实现，在实际生产环境中可以使用更复杂的解析方式
	// 但这里简单处理为不显示完整URL
	return "[已隐藏敏感信息]"
}

func main() {
	Execute()
	//println(int(1 << uint(13)) & 12)
}
