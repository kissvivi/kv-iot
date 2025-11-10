package v1

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"kv-iot/device/endpoint/http/v1/api"
	"time"
)

type engine = *gin.Engine

func InitRouter(api *api.BaseApi) engine {
	r := gin.New()

	// 添加CORS中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 允许所有来源，生产环境应该限制具体域名
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	gin.SetMode("debug")

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong this is kv-iot device endpoint"})
	})
	r.LoadHTMLGlob("static/*")
	r.GET("", func(c *gin.Context) {
		c.HTML(200, "main.html", nil)
	})
	routers(r, api)

	return r
}

func routers(r *gin.Engine, baseApi *api.BaseApi) {
	// 设备基础接口
	deviceGroup := r.Group("/device")
	{
		deviceGroup.POST("", baseApi.ApiDevice.CreateDevice)
		deviceGroup.GET("", baseApi.ApiDevice.GetAllDevice)
		deviceGroup.GET("/by", baseApi.ApiDevice.GetDevice)
		deviceGroup.DELETE("", baseApi.ApiDevice.DelDevice)
		
		// 设备连接管理接口
		connGroup := deviceGroup.Group("/conn")
		{
			connGroup.GET("", baseApi.ApiDeviceConn.GetDeviceConn)            // 获取设备连接信息
			connGroup.GET("/online", baseApi.ApiDeviceConn.GetOnlineDevices)   // 获取在线设备
			connGroup.GET("/all", baseApi.ApiDeviceConn.GetAllDevicesConn)     // 获取所有设备连接
			connGroup.GET("/product", baseApi.ApiDeviceConn.GetProductDevicesConn) // 根据产品获取设备连接
			connGroup.DELETE("", baseApi.ApiDeviceConn.DisconnectDevice)       // 断开设备连接
		}
	}
	
	// 产品基础接口
	productGroup := r.Group("/product")
	{
		productGroup.POST("", baseApi.ApiProduct.CreateProduct)
		productGroup.GET("", baseApi.ApiProduct.GetAllProduct)
		productGroup.GET("/by", baseApi.ApiProduct.GetProduct)
		productGroup.DELETE("", baseApi.ApiProduct.DelProduct)
		
		// 产品物模型接口
		modelGroup := productGroup.Group("/model")
		{
			modelGroup.GET("", baseApi.ApiProductModel.GetProductModel)        // 获取产品物模型
			modelGroup.POST("", baseApi.ApiProductModel.CreateProductModel)    // 创建产品物模型
			modelGroup.GET("/schema", baseApi.ApiProductModel.GetProductSchema) // 获取物模型Schema
			modelGroup.POST("/validate", baseApi.ApiProductModel.ValidateDeviceData) // 验证设备数据
		}
	}
	
	// 通道实例管理接口
	channelGroup := r.Group("/channel")
	{
		channelGroup.POST("/instance", baseApi.ApiChannelInstance.CreateChannelInstance)         // 创建通道实例
		channelGroup.GET("/instance", baseApi.ApiChannelInstance.GetChannelInstances)           // 获取通道实例列表
		channelGroup.GET("/instance/:id", baseApi.ApiChannelInstance.GetChannelInstance)        // 获取通道实例详情
		channelGroup.PUT("/instance/:id", baseApi.ApiChannelInstance.UpdateChannelInstance)     // 更新通道实例
		channelGroup.DELETE("/instance/:id", baseApi.ApiChannelInstance.DeleteChannelInstance)  // 删除通道实例
		channelGroup.POST("/instance/:id/start", baseApi.ApiChannelInstance.StartChannelInstance)    // 启动通道实例
		channelGroup.POST("/instance/:id/stop", baseApi.ApiChannelInstance.StopChannelInstance)      // 停止通道实例
		channelGroup.POST("/instance/:id/restart", baseApi.ApiChannelInstance.RestartChannelInstance) // 重启通道实例
		channelGroup.GET("/instance/:id/status", baseApi.ApiChannelInstance.GetChannelInstanceStatus) // 获取通道实例状态
		channelGroup.PUT("/instance/:id/config", baseApi.ApiChannelInstance.UpdateChannelConfig)      // 更新通道配置
	}
}
