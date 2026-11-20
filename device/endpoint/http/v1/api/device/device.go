package device

import (
	"github.com/gin-gonic/gin"
	"kv-iot/device/data"
	"kv-iot/device/service"
	"kv-iot/pkg/result"
	"strconv"
	"time"
)

type ApiDevice struct {
	baseService *service.BaseService
}

func NewApiDevice(baseService *service.BaseService) *ApiDevice {
	return &ApiDevice{baseService: baseService}
}

func (da ApiDevice) BaseService() *service.BaseService {
	return da.baseService
}

func (da ApiDevice) CreateDevice(c *gin.Context) {
	device := data.Devices{}
	device.LastTime = time.Now()
	c.BindJSON(&device)
	err := da.baseService.DevicesService.AddDevices(device)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "添加失败")
		return
	}
	err, saved := da.baseService.DevicesService.GetDevices(data.Devices{DeviceNo: device.DeviceNo})
	if err == nil && len(saved) > 0 {
		device = saved[0]
		if _, syncErr := da.baseService.GetDevicePointBindingService().SyncForDevice(int64(device.ID)); syncErr != nil {
			_ = da.baseService.DevicesService.DelDevices(device)
			result.BaseResult{}.ErrResult(c, nil, "添加失败: 同步点位绑定失败: "+syncErr.Error())
			return
		}
	}
	result.BaseResult{}.SuccessResult(c, device, "添加成功")

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
	if idStr := c.Query("id"); idStr != "" {
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			result.BaseResult{}.ErrResult(c, nil, "设备ID格式错误")
			return
		}
		device.ID = uint(id)
	}
	device.DeviceNo = c.Query("device_no")
	device.Name = c.Query("name")
	device.ProductKey = c.Query("product_key")
	err, deviceList := da.baseService.DevicesService.GetDevices(device)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败")
	} else {
		result.BaseResult{}.SuccessResult(c, deviceList, "查询成功")
	}

}

func (da ApiDevice) UpdateDevice(c *gin.Context) {
	id, err := parseDeviceID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "设备ID格式错误")
		return
	}

	var update service.DeviceUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误: "+err.Error())
		return
	}

	device, err := da.baseService.DevicesService.UpdateDevice(id, update)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "更新失败: "+err.Error())
		return
	}
	if _, err := da.baseService.GetDevicePointBindingService().SyncForDevice(id); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "更新成功，但同步点位绑定失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, device, "更新成功")
}

func (da ApiDevice) GetAllDevice(c *gin.Context) {
	err, deviceList := da.baseService.DevicesService.GetAllDevices()
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败")
	} else {
		result.BaseResult{}.SuccessResult(c, deviceList, "查询成功")
	}

}

func (da ApiDevice) GetDeviceByID(c *gin.Context) {
	id, err := parseDeviceID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "设备ID格式错误")
		return
	}
	device, err := da.baseService.DevicesService.GetDeviceByID(id)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, device, "查询成功")
}

func (da ApiDevice) DeleteDeviceByID(c *gin.Context) {
	id, err := parseDeviceID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "设备ID格式错误")
		return
	}
	device := data.Devices{}
	device.ID = uint(id)
	if err := da.baseService.DevicesService.DelDevices(device); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "删除失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, nil, "删除成功")
}
