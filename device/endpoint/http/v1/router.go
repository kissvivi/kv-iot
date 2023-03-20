package v1

import (
	"github.com/gin-gonic/gin"
	"kv-iot/device/endpoint/http/v1/api"
)

type engine = *gin.Engine

func InitRouter(api *api.BaseApi) engine {
	r := gin.New()

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
	deviceGroup := r.Group("/device")
	{
		deviceGroup.POST("", baseApi.ApiDevice.CreateDevice)
		deviceGroup.GET("", baseApi.ApiDevice.GetAllDevice)
		deviceGroup.GET("/by", baseApi.ApiDevice.GetDevice)
		deviceGroup.DELETE("", baseApi.ApiDevice.DelDevice)
	}
	productGroup := r.Group("/product")
	{
		productGroup.POST("", baseApi.ApiProduct.CreateProduct)
		productGroup.GET("", baseApi.ApiProduct.GetAllProduct)
		productGroup.GET("/by", baseApi.ApiProduct.GetProduct)
		productGroup.DELETE("", baseApi.ApiProduct.DelProduct)
	}
}
