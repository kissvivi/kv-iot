package device

import (
	"github.com/gin-gonic/gin"
	"kv-iot/device/service"
	"kv-iot/pkg/result"
)

// ApiDeviceConn 设备连接API
type ApiDeviceConn struct {
	baseService *service.BaseService
}

// NewApiDeviceConn 创建设备连接API实例
func NewApiDeviceConn(baseService *service.BaseService) *ApiDeviceConn {
	return &ApiDeviceConn{baseService: baseService}
}

// GetDeviceConn 根据设备编号获取连接信息
func (a *ApiDeviceConn) GetDeviceConn(c *gin.Context) {
	deviceNo := c.Query("device_no")
	if deviceNo == "" {
		result.BaseResult{}.ErrResult(c, nil, "设备编号不能为空")
		return
	}

	err, conn := a.baseService.DeviceConnService.GetConnByDeviceNo(deviceNo)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败: "+err.Error())
		return
	}

	result.BaseResult{}.SuccessResult(c, conn, "查询成功")
}

// GetOnlineDevices 获取在线设备列表
func (a *ApiDeviceConn) GetOnlineDevices(c *gin.Context) {
	err, conns := a.baseService.DeviceConnService.GetOnlineConn()
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败: "+err.Error())
		return
	}

	result.BaseResult{}.SuccessResult(c, conns, "查询成功")
}

// GetAllDevicesConn 获取所有设备连接信息
func (a *ApiDeviceConn) GetAllDevicesConn(c *gin.Context) {
	err, conns := a.baseService.DeviceConnService.GetAllConn()
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败: "+err.Error())
		return
	}

	result.BaseResult{}.SuccessResult(c, conns, "查询成功")
}

// GetProductDevicesConn 根据产品标识获取设备连接信息
func (a *ApiDeviceConn) GetProductDevicesConn(c *gin.Context) {
	productKey := c.Query("product_key")
	if productKey == "" {
		result.BaseResult{}.ErrResult(c, nil, "产品标识不能为空")
		return
	}

	err, conns := a.baseService.DeviceConnService.GetConnByProductKey(productKey)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败: "+err.Error())
		return
	}

	result.BaseResult{}.SuccessResult(c, conns, "查询成功")
}

// DisconnectDevice 断开设备连接
func (a *ApiDeviceConn) DisconnectDevice(c *gin.Context) {
	deviceNo := c.Query("device_no")
	if deviceNo == "" {
		result.BaseResult{}.ErrResult(c, nil, "设备编号不能为空")
		return
	}

	err := a.baseService.DeviceConnService.Disconnect(deviceNo)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "断开连接失败: "+err.Error())
		return
	}

	result.BaseResult{}.SuccessResult(c, nil, "断开连接成功")
}