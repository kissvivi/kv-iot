package service

import (
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
)

var _ DevicesService = (*DevicesServiceImpl)(nil)

type DevicesService interface {
	AddDevices(devices data.Devices) (err error)
	DelDevices(devices data.Devices) (err error)
}

type DevicesServiceImpl struct {
	devices repo.DevicesRepo
}

func (d DevicesServiceImpl) AddDevices(devices data.Devices) (err error) {
	return d.devices.Add(devices)
}

func (d DevicesServiceImpl) DelDevices(devices data.Devices) (err error) {
	return d.devices.Delete(devices)
}
