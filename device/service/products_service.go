package service

import (
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
)

var _ ProductsService = (*ProductsServiceImpl)(nil)

type ProductsService interface {
	AddProducts(products data.Products) (err error)
	DelProducts(products data.Products) (err error)
	GetProducts(products data.Products) (err error, productsList []data.Products)
	GetAllProducts() (err error, productsList []data.Products)
}

type ProductsServiceImpl struct {
	pd repo.ProductsRepo
}

func NewProductsServiceImpl(pd repo.ProductsRepo) *ProductsServiceImpl {
	return &ProductsServiceImpl{pd: pd}
}

func (pd ProductsServiceImpl) AddProducts(products data.Products) (err error) {
	return pd.pd.Add(products)
}

func (pd ProductsServiceImpl) DelProducts(products data.Products) (err error) {
	return pd.pd.Delete(products)
}

func (pd ProductsServiceImpl) GetAllProducts() (err error, productsList []data.Products) {
	err, productsList = pd.pd.FindAll()
	return
}

func (pd ProductsServiceImpl) GetProducts(products data.Products) (err error, productsList []data.Products) {
	err, productsList = pd.pd.FindByStruct(products)
	return
}
