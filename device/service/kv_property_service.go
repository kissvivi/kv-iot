package service

import (
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
)

var _ KvPropertyService = (*KvPropertyServiceImpl)(nil)

// KvPropertyService 属性服务接口
type KvPropertyService interface {
	AddKvProperty(property data.KvProperty) (err error)
	DelKvProperty(property data.KvProperty) (err error)
	UpdateKvProperty(property data.KvProperty) (err error)
	GetKvProperty(property data.KvProperty) (err error, propertyList []data.KvProperty)
	GetAllKvProperty() (err error, propertyList []data.KvProperty)
	GetPropertyByProductID(productID int64) (err error, propertyList []data.KvProperty)
	ValidatePropertyValue(property data.KvProperty, value interface{}) (bool, error)
}

// KvPropertyServiceImpl 属性服务实现
type KvPropertyServiceImpl struct {
	property repo.KvPropertyRepo
}

// NewKvPropertyServiceImpl 创建属性服务实例
func NewKvPropertyServiceImpl(property repo.KvPropertyRepo) *KvPropertyServiceImpl {
	return &KvPropertyServiceImpl{property: property}
}

// AddKvProperty 添加属性
func (kp KvPropertyServiceImpl) AddKvProperty(property data.KvProperty) (err error) {
	return kp.property.Add(property)
}

// DelKvProperty 删除属性
func (kp KvPropertyServiceImpl) DelKvProperty(property data.KvProperty) (err error) {
	return kp.property.Delete(property)
}

// UpdateKvProperty 更新属性
func (kp KvPropertyServiceImpl) UpdateKvProperty(property data.KvProperty) (err error) {
	return kp.property.Update(property)
}

// GetAllKvProperty 获取所有属性
func (kp KvPropertyServiceImpl) GetAllKvProperty() (err error, propertyList []data.KvProperty) {
	err, propertyList = kp.property.FindAll()
	return
}

// GetKvProperty 根据条件查询属性
func (kp KvPropertyServiceImpl) GetKvProperty(property data.KvProperty) (err error, propertyList []data.KvProperty) {
	err, propertyList = kp.property.FindByStruct(property)
	return
}

// GetPropertyByProductID 根据产品ID查询属性
func (kp KvPropertyServiceImpl) GetPropertyByProductID(productID int64) (err error, propertyList []data.KvProperty) {
	property := data.KvProperty{}
	property.ProductID = productID
	return kp.GetKvProperty(property)
}

// ValidatePropertyValue 验证属性值
func (kp KvPropertyServiceImpl) ValidatePropertyValue(property data.KvProperty, value interface{}) (bool, error) {
	// 这里可以实现根据数据类型验证属性值的逻辑
	// 目前简单返回true，后续可以扩展更复杂的验证
	return true, nil
}