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

// serverCmd represents the endpoint command
var serverCmd = &cobra.Command{
	Use:   "run",
	Short: "kv-iot",
	Long:  `kv-iot`,
	Run:   runServer,
}

//type Application struct {
//	name string
//	version string
//	httpServer *rest.Server
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
	go initGrpcServer()
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
	fmt.Printf("%+v\n", cfg)
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
		//logrus.Error(err)
		fmt.Errorf("%s", err)
		os.Exit(1)
	}
}

func main() {
	Execute()
	//println(int(1 << uint(13)) & 12)
}
