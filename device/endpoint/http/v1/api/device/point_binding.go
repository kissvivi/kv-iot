package device

import (
	"kv-iot/device/data"
	"kv-iot/device/service"
	"kv-iot/pkg/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ApiDevicePointBinding struct {
	baseService *service.BaseService
}

func NewApiDevicePointBinding(baseService *service.BaseService) *ApiDevicePointBinding {
	return &ApiDevicePointBinding{baseService: baseService}
}

func (a *ApiDevicePointBinding) ListByDevice(c *gin.Context) {
	deviceID, err := parseDeviceID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "设备ID格式错误")
		return
	}
	bindings, err := a.baseService.GetDevicePointBindingService().ListByDevice(deviceID)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询设备点位绑定失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, bindings, "查询成功")
}

func (a *ApiDevicePointBinding) SyncForDevice(c *gin.Context) {
	deviceID, err := parseDeviceID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "设备ID格式错误")
		return
	}
	created, err := a.baseService.GetDevicePointBindingService().SyncForDevice(deviceID)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "同步设备点位绑定失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, created, "同步成功")
}

func (a *ApiDevicePointBinding) Create(c *gin.Context) {
	deviceID, err := parseDeviceID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "设备ID格式错误")
		return
	}
	var binding data.DevicePointBinding
	if err := c.ShouldBindJSON(&binding); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误: "+err.Error())
		return
	}
	created, err := a.baseService.GetDevicePointBindingService().Create(deviceID, binding)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "创建设备点位绑定失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, created, "创建成功")
}

func (a *ApiDevicePointBinding) Get(c *gin.Context) {
	id, err := parseDeviceID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "点位绑定ID格式错误")
		return
	}
	binding, err := a.baseService.GetDevicePointBindingService().GetByID(id)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询设备点位绑定失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, binding, "查询成功")
}

func (a *ApiDevicePointBinding) Update(c *gin.Context) {
	id, err := parseDeviceID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "点位绑定ID格式错误")
		return
	}
	var update service.DevicePointBindingUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误: "+err.Error())
		return
	}
	binding, err := a.baseService.GetDevicePointBindingService().Update(id, update)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "更新设备点位绑定失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, binding, "更新成功")
}

func (a *ApiDevicePointBinding) Delete(c *gin.Context) {
	id, err := parseDeviceID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "点位绑定ID格式错误")
		return
	}
	if err := a.baseService.GetDevicePointBindingService().Delete(id); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "删除设备点位绑定失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, nil, "删除成功")
}

func parseDeviceID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}
