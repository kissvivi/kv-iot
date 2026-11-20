package product

import (
	"github.com/gin-gonic/gin"
	"kv-iot/device/data"
	"kv-iot/device/service"
	"kv-iot/pkg/result"
	"strconv"
)

type ApiProduct struct {
	baseService *service.BaseService
}

func NewApiProduct(baseService *service.BaseService) *ApiProduct {
	return &ApiProduct{baseService: baseService}
}

func (da ApiProduct) BaseService() *service.BaseService {
	return da.baseService
}

func (da ApiProduct) CreateProduct(c *gin.Context) {
	product := data.Products{}
	c.BindJSON(&product)
	err := da.baseService.ProductsService.AddProducts(product)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "添加失败")

	} else {
		result.BaseResult{}.SuccessResult(c, product, "添加成功")
	}

}

func (da ApiProduct) DelProduct(c *gin.Context) {
	product := data.Products{}
	c.BindJSON(&product)
	err := da.baseService.ProductsService.DelProducts(product)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "删除失败")
	} else {
		result.BaseResult{}.SuccessResult(c, nil, "删除成功")
	}

}

func (da ApiProduct) GetProduct(c *gin.Context) {
	product := data.Products{}
	product.Name = c.Query("name")
	product.ProductKey = c.Query("product_key")
	product.ChannelID = c.Query("channel_id")
	err, productList := da.baseService.ProductsService.GetProducts(product)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败")
	} else {
		result.BaseResult{}.SuccessResult(c, productList, "查询成功")
	}

}

func (da ApiProduct) UpdateProduct(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "产品ID格式错误")
		return
	}

	product := data.Products{}
	if err := c.ShouldBindJSON(&product); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误")
		return
	}
	product.ID = uint(id)

	if err := da.baseService.ProductsService.UpdateProducts(product); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "更新失败")
		return
	}
	result.BaseResult{}.SuccessResult(c, product, "更新成功")
}

func (da ApiProduct) GetAllProduct(c *gin.Context) {
	err, productList := da.baseService.ProductsService.GetAllProducts()
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败")
	} else {
		result.BaseResult{}.SuccessResult(c, productList, "查询成功")
	}

}

func (da ApiProduct) GetProductByID(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "产品ID格式错误")
		return
	}
	product, err := da.baseService.ProductsService.GetProductByID(id)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, product, "查询成功")
}

func (da ApiProduct) DeleteProductByID(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "产品ID格式错误")
		return
	}
	product := data.Products{}
	product.ID = uint(id)
	if err := da.baseService.ProductsService.DelProducts(product); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "删除失败: "+err.Error())
		return
	}
	result.BaseResult{}.SuccessResult(c, nil, "删除成功")
}
