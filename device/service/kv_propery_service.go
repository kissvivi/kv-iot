package service

import (
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
)

var _ KvPropertyService = (*KvPropertyServiceImpl)(nil)

type KvPropertyService interface {
	AddKvProperty(property data.KvProperty) (err error)
	DelKvProperty(property data.KvProperty) (err error)
	GetKvProperty(property data.KvProperty) (err error, propertyList []data.KvProperty)
	GetAllKvProperty() (err error, propertyList []data.KvProperty)
}

type KvPropertyServiceImpl struct {
	property repo.KvPropertyRepo
}

func NewKvPropertyServiceImpl(property repo.KvPropertyRepo) *KvPropertyServiceImpl {
	return &KvPropertyServiceImpl{property: property}
}

func (kp KvPropertyServiceImpl) AddKvProperty(property data.KvProperty) (err error) {
	return kp.property.Add(property)
}

func (kp KvPropertyServiceImpl) DelKvProperty(property data.KvProperty) (err error) {
	return kp.property.Delete(property)
}

func (kp KvPropertyServiceImpl) GetAllKvProperty() (err error, deviceList []data.KvProperty) {
	err, deviceList = kp.property.FindAll()
	return
}

func (kp KvPropertyServiceImpl) GetKvProperty(property data.KvProperty) (err error, deviceList []data.KvProperty) {
	err, deviceList = kp.property.FindByStruct(property)
	return
}
