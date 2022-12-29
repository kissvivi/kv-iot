package v1

import (
	"github.com/gin-gonic/gin"
	"kv-iot/auth/endpoint/http/v1/api"
)

type engine = *gin.Engine

func InitRouter(api api.BaseApi) engine {
	r := gin.New()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	gin.SetMode("debug")

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong this is kv-iot auth endpoint"})
	})

	routers(r, api)

	return r
}

func routers(r *gin.Engine, baseApi api.BaseApi) {
	//g := r.Group("/test")
	//{
	//	//g.GET("/ok", baseApi)
	//	//g.GET("/demo/ok", baseApi.DemoOk)
	//	//g.POST("/create", baseApi.CreateTest)
	//}
}
