package product

import (
	"github.com/gin-gonic/gin"
	"kv-iot/device/data"
	"kv-iot/device/service"
	"kv-iot/pkg/result"
)

type ApiProduct struct {
	baseService *service.BaseService
}

func NewApiProduct(baseService *service.BaseService) *ApiProduct {
	return &ApiProduct{baseService: baseService}
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
	c.BindJSON(&product)
	err, productList := da.baseService.ProductsService.GetProducts(product)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败")
	} else {
		result.BaseResult{}.SuccessResult(c, productList, "查询成功")
	}

}

func (da ApiProduct) GetAllProduct(c *gin.Context) {
	err, productList := da.baseService.ProductsService.GetAllProducts()
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败")
	} else {
		result.BaseResult{}.SuccessResult(c, productList, "查询成功")
	}

}
