package product

import (
	"kv-iot/device/data"
	"kv-iot/device/service"
	"kv-iot/pkg/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ApiProductProtocolProfile struct {
	baseService *service.BaseService
}

func NewApiProductProtocolProfile(baseService *service.BaseService) *ApiProductProtocolProfile {
	return &ApiProductProtocolProfile{baseService: baseService}
}

func (a *ApiProductProtocolProfile) ListByProduct(c *gin.Context) {
	productID, err := parseID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "产品ID格式错误")
		return
	}
	profiles, err := a.baseService.GetProductProtocolProfileService().ListByProduct(productID)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询协议版本失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, profiles, "查询成功")
}

func (a *ApiProductProtocolProfile) Create(c *gin.Context) {
	productID, err := parseID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "产品ID格式错误")
		return
	}
	var profile data.ProductProtocolProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误: "+err.Error())
		return
	}
	created, err := a.baseService.GetProductProtocolProfileService().Create(productID, profile)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "创建协议版本失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, created, "创建成功")
}

func (a *ApiProductProtocolProfile) Get(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "协议版本ID格式错误")
		return
	}
	profile, err := a.baseService.GetProductProtocolProfileService().GetByID(id)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询协议版本失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, profile, "查询成功")
}

func (a *ApiProductProtocolProfile) Update(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "协议版本ID格式错误")
		return
	}
	var update service.ProductProtocolProfileUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误: "+err.Error())
		return
	}
	profile, err := a.baseService.GetProductProtocolProfileService().Update(id, update)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "更新协议版本失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, profile, "更新成功")
}

func (a *ApiProductProtocolProfile) Delete(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "协议版本ID格式错误")
		return
	}
	if err := a.baseService.GetProductProtocolProfileService().Delete(id); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "删除协议版本失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, nil, "删除成功")
}

func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}
