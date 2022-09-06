package baseapi

import (
	"github.com/gin-gonic/gin"
	"github.kissvivi.kv-iot/device/data"
	"github.kissvivi.kv-iot/device/service"
)

type DeviceApi struct {
	baseService service.BaseServiceImpl
}

func (d DeviceApi) TestDevice(c *gin.Context) {

	d.baseService.AddDevice(data.Devices{Name: "test"})
}
