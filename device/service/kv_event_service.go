package service

import (
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
)

var _ KvEventService = (*KvEventServiceImpl)(nil)

type KvEventService interface {
	AddKvEvent(event data.KvEvent) (err error)
	DelKvEvent(event data.KvEvent) (err error)
	GetKvEvent(event data.KvEvent) (err error, eventList []data.KvEvent)
	GetAllKvEvent() (err error, eventList []data.KvEvent)
}

type KvEventServiceImpl struct {
	event repo.KvEventRepo
}

func NewKvEventServiceImpl(event repo.KvEventRepo) *KvEventServiceImpl {
	return &KvEventServiceImpl{event: event}
}

func (ke KvEventServiceImpl) AddKvEvent(event data.KvEvent) (err error) {
	return ke.event.Add(event)
}

func (ke KvEventServiceImpl) DelKvEvent(event data.KvEvent) (err error) {
	return ke.event.Delete(event)
}

func (ke KvEventServiceImpl) GetAllKvEvent() (err error, deviceList []data.KvEvent) {
	err, deviceList = ke.event.FindAll()
	return
}

func (ke KvEventServiceImpl) GetKvEvent(event data.KvEvent) (err error, deviceList []data.KvEvent) {
	err, deviceList = ke.event.FindByStruct(event)
	return
}
