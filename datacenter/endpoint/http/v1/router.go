package v1

import (
	"github.com/gin-gonic/gin"

	authservice "kv-iot/auth/service"
	"kv-iot/datacenter/runtime"
)

func NewRouter(api *API) *gin.Engine {
	return NewRouterWithMiddlewares(api, authservice.AuthRequired())
}

// NewRouterWithMiddlewares is useful for tests and for deployments that
// terminate authentication at a trusted gateway.
func NewRouterWithMiddlewares(api *API, middlewares ...gin.HandlerFunc) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = runtime.GenerateRequestID()
		}
		c.Set("runtime_request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	})
	runtimeGroup := router.Group("/runtime")
	api.RegisterPublicRoutes(router)
	runtimeGroup.Use(middlewares...)
	api.RegisterRoutes(runtimeGroup)
	return router
}
