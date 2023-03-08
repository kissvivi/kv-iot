package service

import (
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
)

var _ KvEventService = (*KvEventServiceImpl)(nil)

type KvEventService interface {
	AddKvEvent(kvEvent data.KvEvent) (err error)
	DelKvEvent(kvEvent data.KvEvent) (err error)
}

type KvEventServiceImpl struct {
	event repo.KvEventRepo
}

func NewKvEventServiceImpl(event repo.KvEventRepo) *KvEventServiceImpl {
	return &KvEventServiceImpl{event: event}
}

func (a KvEventServiceImpl) AddKvEvent(kvEvent data.KvEvent) (err error) {
	return a.event.Add(kvEvent)
}

func (a KvEventServiceImpl) DelKvEvent(kvEvent data.KvEvent) (err error) {
	return a.event.Delete(kvEvent)
}
