package service

import (
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
)

var _ KvActionService = (*KvActionServiceImpl)(nil)

type KvActionService interface {
	AddKvAction(kvAction data.KvAction) (err error)
	DelKvAction(kvAction data.KvAction) (err error)
	GetKvAction(action data.KvAction) (err error, actionList []data.KvAction)
	GetAllKvAction() (err error, actionList []data.KvAction)
}

type KvActionServiceImpl struct {
	action repo.KvActionRepo
}

func NewKvActionServiceImpl(action repo.KvActionRepo) *KvActionServiceImpl {
	return &KvActionServiceImpl{action: action}
}

func (ka KvActionServiceImpl) AddKvAction(action data.KvAction) (err error) {
	return ka.action.Add(action)
}

func (ka KvActionServiceImpl) DelKvAction(action data.KvAction) (err error) {
	return ka.action.Delete(action)
}

func (ka KvActionServiceImpl) GetAllKvAction() (err error, deviceList []data.KvAction) {
	err, deviceList = ka.action.FindAll()
	return
}

func (ka KvActionServiceImpl) GetKvAction(action data.KvAction) (err error, deviceList []data.KvAction) {
	err, deviceList = ka.action.FindByStruct(action)
	return
}
