package device

import (
	"github.com/gin-gonic/gin"
	"kv-iot/device/data"
	"kv-iot/device/service"
	"kv-iot/pkg/result"
	"time"
)

type ApiDevice struct {
	baseService *service.BaseService
}

func NewApiDevice(baseService *service.BaseService) *ApiDevice {
	return &ApiDevice{baseService: baseService}
}

func (da ApiDevice) CreateDevice(c *gin.Context) {
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

func (da ApiDevice) DelDevice(c *gin.Context) {
	device := data.Devices{}
	device.LastTime = time.Now()
	c.BindJSON(&device)
	err := da.baseService.DevicesService.DelDevices(device)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "删除失败")
	} else {
		result.BaseResult{}.SuccessResult(c, nil, "删除成功")
	}

}

func (da ApiDevice) GetDevice(c *gin.Context) {
	device := data.Devices{}
	device.LastTime = time.Now()
	c.BindJSON(&device)
	err, deviceList := da.baseService.DevicesService.GetDevices(device)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败")
	} else {
		result.BaseResult{}.SuccessResult(c, deviceList, "查询成功")
	}

}

func (da ApiDevice) GetAllDevice(c *gin.Context) {
	err, deviceList := da.baseService.DevicesService.GetAllDevices()
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败")
	} else {
		result.BaseResult{}.SuccessResult(c, deviceList, "查询成功")
	}

}
