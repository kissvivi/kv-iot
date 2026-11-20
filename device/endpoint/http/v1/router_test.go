package v1

import (
	"testing"

	"github.com/gin-gonic/gin"
	"kv-iot/device/endpoint/http/v1/api"
	"kv-iot/device/endpoint/http/v1/api/channel"
	deviceapi "kv-iot/device/endpoint/http/v1/api/device"
	productapi "kv-iot/device/endpoint/http/v1/api/product"
	"kv-iot/device/service"
)

func TestDeviceRoutesRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	baseService := service.NewBaseService(nil, nil, nil, nil, nil, nil)
	deviceApi := deviceapi.NewApiDevice(baseService)
	productApi := productapi.NewApiProduct(baseService)
	baseApi := api.NewBaseApi(deviceApi, productApi)
	baseApi.SetApiDeviceConn(deviceapi.NewApiDeviceConn(baseService))
	baseApi.SetApiProductModel(productapi.NewApiProductModel(baseService))
	baseApi.SetApiChannelInstance(channel.NewApiChannelInstance(baseService))

	router := gin.New()
	routers(router, baseApi)

	want := map[string]bool{
		"GET /device/:id":                   false,
		"PUT /device/:id":                   false,
		"GET /device/:id/point-bindings":    false,
		"POST /device/:id/point-bindings":   false,
		"GET /device/point-bindings/:id":    false,
		"GET /product/:id/model":            false,
		"PUT /product/:id/model":            false,
		"DELETE /product/:id/model":         false,
		"GET /product/:id/profiles":         false,
		"POST /product/:id/profiles":        false,
		"GET /profiles/:id":                 false,
		"PUT /profiles/:id":                 false,
		"DELETE /profiles/:id":              false,
		"GET /profiles/:id/point-bindings":  false,
		"POST /profiles/:id/point-bindings": false,
		"POST /point-bindings/validate":     false,
		"GET /point-bindings/:id":           false,
		"PUT /point-bindings/:id":           false,
		"DELETE /point-bindings/:id":        false,
	}
	for _, route := range router.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("route %s was not registered", route)
		}
	}
}
