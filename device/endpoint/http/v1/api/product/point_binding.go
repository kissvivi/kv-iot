package product

import (
	"kv-iot/device/data"
	"kv-iot/device/service"
	"kv-iot/pkg/result"

	"github.com/gin-gonic/gin"
)

type ApiProductPointBinding struct {
	baseService *service.BaseService
}

func NewApiProductPointBinding(baseService *service.BaseService) *ApiProductPointBinding {
	return &ApiProductPointBinding{baseService: baseService}
}

func (a *ApiProductPointBinding) ListByProfile(c *gin.Context) {
	profileID, err := parseID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "协议版本ID格式错误")
		return
	}
	bindings, err := a.baseService.GetProductPointBindingService().ListByProfile(profileID)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询点位绑定失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, bindings, "查询成功")
}

func (a *ApiProductPointBinding) Create(c *gin.Context) {
	profileID, err := parseID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "协议版本ID格式错误")
		return
	}
	var binding data.ProductPointBinding
	if err := c.ShouldBindJSON(&binding); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误: "+err.Error())
		return
	}
	created, err := a.baseService.GetProductPointBindingService().Create(profileID, binding)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "创建点位绑定失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, created, "创建成功")
}

func (a *ApiProductPointBinding) Get(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "点位绑定ID格式错误")
		return
	}
	binding, err := a.baseService.GetProductPointBindingService().GetByID(id)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询点位绑定失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, binding, "查询成功")
}

func (a *ApiProductPointBinding) Update(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "点位绑定ID格式错误")
		return
	}
	var update service.ProductPointBindingUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误: "+err.Error())
		return
	}
	binding, err := a.baseService.GetProductPointBindingService().Update(id, update)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "更新点位绑定失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, binding, "更新成功")
}

func (a *ApiProductPointBinding) Delete(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "点位绑定ID格式错误")
		return
	}
	if err := a.baseService.GetProductPointBindingService().Delete(id); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "删除点位绑定失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, nil, "删除成功")
}

func (a *ApiProductPointBinding) Validate(c *gin.Context) {
	var request struct {
		ProtocolType  string `json:"protocol_type"`
		BindingConfig string `json:"binding_config"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误: "+err.Error())
		return
	}
	validation := a.baseService.GetProductPointBindingService().Validate(request.ProtocolType, request.BindingConfig)
	result.BaseResult{}.SuccessResult(c, validation, "校验完成")
}
