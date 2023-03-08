package baseapi

import (
	"github.com/gin-gonic/gin"
	"kv-iot/device/data"
	"kv-iot/device/service"
	"kv-iot/pkg/result"
	"time"
)

type DeviceApi struct {
	baseService *service.BaseService
}

func NewDeviceApi(baseService *service.BaseService) *DeviceApi {
	return &DeviceApi{baseService: baseService}
}

func (da DeviceApi) CreateDevice(c *gin.Context) {
	device := data.Devices{}
	device.LastTime = time.Now()
	c.BindJSON(&device)
	err := da.baseService.DevicesService.AddDevices(device)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "添加失败")

	} else {
		result.BaseResult{}.SuccessResult(c, device, "添加成功")
	}

}
