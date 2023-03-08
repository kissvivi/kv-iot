package service

import (
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
)

var _ KvPropertyService = (*KvPropertyServiceImpl)(nil)

type KvPropertyService interface {
	AddKvProperty(kvProperty data.KvProperty) (err error)
	DelKvProperty(kvProperty data.KvProperty) (err error)
}

type KvPropertyServiceImpl struct {
	property repo.KvPropertyRepo
}

func NewKvPropertyServiceImpl(property repo.KvPropertyRepo) *KvPropertyServiceImpl {
	return &KvPropertyServiceImpl{property: property}
}

func (a KvPropertyServiceImpl) AddKvProperty(kvProperty data.KvProperty) (err error) {
	return a.property.Add(kvProperty)
}

func (a KvPropertyServiceImpl) DelKvProperty(kvProperty data.KvProperty) (err error) {
	return a.property.Delete(kvProperty)
}
