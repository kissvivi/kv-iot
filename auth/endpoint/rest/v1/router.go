package v1

import (
	"github.com/gin-gonic/gin"
	"kv-iot/auth/endpoint/rest"
	"kv-iot/auth/endpoint/rest/v1/api"
	"kv-iot/auth/service"
	"log"
)

type engine = *gin.Engine

func InitRouter(api *api.BaseApi) engine {
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

func routers(r *gin.Engine, baseApi *api.BaseApi) {
	g := r.Group("")
	{
		g.POST("/login", baseApi.UserApi.Login)
	}
	g1 := r.Group("/auth")
	g1.Use(JWTAuth())
	{
		g1.POST("/login1", baseApi.UserApi.Login)
	}
}

// JWTAuth 定义一个JWTAuth的中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 通过http header中的token解析来认证
		token := c.Request.Header.Get("token")
		if token == "" {
			rest.BaseResult{}.NoTokenResult(c, nil, "请求未携带token，无权限访问")
			c.Abort()
			return
		}

		log.Print("get token: ", token)

		// 解析token中包含的相关信息(有效载荷)
		claims, err := service.ParseToken(token)

		if err != nil {
			// token过期
			if err == service.ErrorTokenExpired {
				rest.BaseResult{}.NoTokenResult(c, nil, "token授权已过期，请重新申请授权")
				c.Abort()
				return
			}
			// 其他错误
			rest.BaseResult{}.ErrResult(c, nil, err.Error())
			c.Abort()
			return
		}

		// 将解析后的有效载荷claims重新写入gin.Context引用对象中
		c.Set("claims", claims)

	}
}
