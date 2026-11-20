package v1

import (
	"github.com/gin-gonic/gin"
	"kv-iot/auth/endpoint/rest/v1/api"
	"kv-iot/auth/service"
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
	r.LoadHTMLGlob("static/*")
	r.GET("", func(c *gin.Context) {
		c.HTML(200, "main.html", nil)
	})

	routers(r, api)

	return r
}

func routers(r *gin.Engine, baseApi *api.BaseApi) {
	r.POST("/login", baseApi.UserApi.Login)
	r.POST("/reg", baseApi.UserApi.RegUser)
	r.POST("/auth/login1", baseApi.UserApi.Login)

	apiV1 := r.Group("/api/v1")
	apiV1.Use(service.AuthRequired())
	{
		apiV1.GET("/me", baseApi.UserApi.Me)

		admin := apiV1.Group("")
		admin.Use(service.PermissionRequired("system:manage"))
		{
			admin.GET("/users", baseApi.UserApi.ListUsers)
			admin.POST("/users", baseApi.UserApi.CreateUser)
			admin.GET("/users/:id", baseApi.UserApi.GetUser)
			admin.PUT("/users/:id", baseApi.UserApi.UpdateUser)
			admin.DELETE("/users/:id", baseApi.UserApi.DeleteUser)

			admin.GET("/roles", baseApi.RoleApi.ListRoles)
			admin.POST("/roles", baseApi.RoleApi.CreateRole)
			admin.GET("/roles/:id", baseApi.RoleApi.GetRole)
			admin.PUT("/roles/:id", baseApi.RoleApi.UpdateRole)
			admin.DELETE("/roles/:id", baseApi.RoleApi.DeleteRole)
			admin.GET("/roles/:id/permissions", baseApi.RoleApi.GetRolePermissions)
			admin.POST("/roles/:id/permissions", baseApi.RoleApi.AssignRolePermissions)

			admin.GET("/modules", baseApi.ModuleApi.ListModules)
			admin.POST("/modules", baseApi.ModuleApi.CreateModule)
			admin.GET("/modules/:id", baseApi.ModuleApi.GetModule)
			admin.PUT("/modules/:id", baseApi.ModuleApi.UpdateModule)
			admin.DELETE("/modules/:id", baseApi.ModuleApi.DeleteModule)

			admin.GET("/permissions", baseApi.PermissionApi.ListPermissions)
			admin.POST("/permissions", baseApi.PermissionApi.CreatePermission)
			admin.GET("/permissions/:id", baseApi.PermissionApi.GetPermission)
			admin.PUT("/permissions/:id", baseApi.PermissionApi.UpdatePermission)
			admin.DELETE("/permissions", baseApi.PermissionApi.DeleteRoleModulePermission)
			admin.DELETE("/permissions/:id", baseApi.PermissionApi.DeletePermission)
		}
	}
}

// JWTAuth remains for callers that imported the old router helper. New code
// should use service.AuthRequired directly.
func JWTAuth() gin.HandlerFunc {
	return service.AuthRequired()
}
