package product

import (
	"github.com/gin-gonic/gin"
	"kv-iot/device/service"
	"kv-iot/pkg/result"
	"strconv"
)

// ApiProductModel 产品物模型API
type ApiProductModel struct {
	baseService *service.BaseService
}

// NewApiProductModel 创建产品物模型API实例
func NewApiProductModel(baseService *service.BaseService) *ApiProductModel {
	return &ApiProductModel{baseService: baseService}
}

// GetProductModel 获取产品完整物模型
func (a *ApiProductModel) GetProductModel(c *gin.Context) {
	productIDStr := c.Query("product_id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "产品ID格式错误")
		return
	}

	err, model := a.baseService.ProductModelService.GetProductModel(productID)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "获取物模型失败: "+err.Error())
		return
	}

	result.BaseResult{}.SuccessResult(c, model, "获取物模型成功")
}

// CreateProductModel 创建产品物模型
func (a *ApiProductModel) CreateProductModel(c *gin.Context) {
	productIDStr := c.Query("product_id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "产品ID格式错误")
		return
	}

	var model service.ProductModel
	if err := c.ShouldBindJSON(&model); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "请求参数格式错误: "+err.Error())
		return
	}

	err = a.baseService.ProductModelService.CreateProductModel(productID, model)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "创建物模型失败: "+err.Error())
		return
	}

	result.BaseResult{}.SuccessResult(c, nil, "创建物模型成功")
}

// GetProductSchema 获取产品物模型JSON Schema
func (a *ApiProductModel) GetProductSchema(c *gin.Context) {
	productIDStr := c.Query("product_id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "产品ID格式错误")
		return
	}

	err, schema := a.baseService.ProductModelService.GetProductSchema(productID)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "获取Schema失败: "+err.Error())
		return
	}

	result.BaseResult{}.SuccessResult(c, schema, "获取Schema成功")
}

// ValidateDeviceData 验证设备数据
func (a *ApiProductModel) ValidateDeviceData(c *gin.Context) {
	productIDStr := c.Query("product_id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "产品ID格式错误")
		return
	}

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "请求参数格式错误: "+err.Error())
		return
	}

	valid, err := a.baseService.ProductModelService.ValidateDeviceData(productID, data)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "验证失败: "+err.Error())
		return
	}

	if valid {
		result.BaseResult{}.SuccessResult(c, nil, "数据验证通过")
	} else {
		result.BaseResult{}.ErrResult(c, nil, "数据验证失败")
	}
}