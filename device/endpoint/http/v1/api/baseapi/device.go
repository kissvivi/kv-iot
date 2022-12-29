package baseapi

import (
	"github.com/gin-gonic/gin"
	"kv-iot/device/data"
	"kv-iot/device/service"
)

type DeviceApi struct {
	baseService service.BaseServiceImpl
}

func (d DeviceApi) TestDevice(c *gin.Context) {

	d.baseService.AddDevice(data.Devices{Name: "test"})
}
