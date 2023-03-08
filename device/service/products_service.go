package service

import (
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
)

var _ ProductsService = (*ProductsServiceImpl)(nil)

type ProductsService interface {
	AddProducts(products data.Products) (err error)
	DelProducts(products data.Products) (err error)
}

type ProductsServiceImpl struct {
	pd repo.ProductsRepo
}

func NewProductsServiceImpl(pd repo.ProductsRepo) *ProductsServiceImpl {
	return &ProductsServiceImpl{pd: pd}
}

func (a ProductsServiceImpl) AddProducts(products data.Products) (err error) {
	return a.pd.Add(products)
}

func (a ProductsServiceImpl) DelProducts(products data.Products) (err error) {
	return a.pd.Delete(products)
}
