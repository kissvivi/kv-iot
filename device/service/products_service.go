package service

import (
	"errors"
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
	"strings"

	"gorm.io/gorm"
)

var _ ProductsService = (*ProductsServiceImpl)(nil)

type ProductsService interface {
	AddProducts(products data.Products) (err error)
	UpdateProducts(products data.Products) (err error)
	DelProducts(products data.Products) (err error)
	GetProducts(products data.Products) (err error, productsList []data.Products)
	GetAllProducts() (err error, productsList []data.Products)
	GetProductByID(id int64) (*data.Products, error)
}

type ProductsServiceImpl struct {
	pd repo.ProductsRepo
}

func NewProductsServiceImpl(pd repo.ProductsRepo) *ProductsServiceImpl {
	return &ProductsServiceImpl{pd: pd}
}

func (pd ProductsServiceImpl) AddProducts(products data.Products) (err error) {
	products.ProductKey = strings.TrimSpace(products.ProductKey)
	if products.ProductKey == "" {
		return errors.New("product_key is required")
	}
	err, existing := pd.GetProducts(data.Products{ProductKey: products.ProductKey})
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return errors.New("product_key already exists")
	}
	return pd.pd.Add(products)
}

func (pd ProductsServiceImpl) UpdateProducts(products data.Products) (err error) {
	return pd.pd.Update(products)
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

func (pd ProductsServiceImpl) GetProductByID(id int64) (*data.Products, error) {
	err, products := pd.GetProducts(data.Products{Model: gorm.Model{ID: uint(id)}})
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &products[0], nil
}
