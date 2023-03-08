package service

import (
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
)

var _ KvActionService = (*KvActionServiceImpl)(nil)

type KvActionService interface {
	AddKvAction(kvAction data.KvAction) (err error)
	DelKvAction(kvAction data.KvAction) (err error)
}

type KvActionServiceImpl struct {
	action repo.KvActionRepo
}

func NewKvActionServiceImpl(action repo.KvActionRepo) *KvActionServiceImpl {
	return &KvActionServiceImpl{action: action}
}

func (a KvActionServiceImpl) AddKvAction(kvAction data.KvAction) (err error) {
	return a.action.Add(kvAction)
}

func (a KvActionServiceImpl) DelKvAction(kvAction data.KvAction) (err error) {
	return a.action.Delete(kvAction)
}
