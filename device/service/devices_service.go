package service

import (
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
)

var _ DevicesService = (*DevicesServiceImpl)(nil)

type DevicesService interface {
	AddDevices(devices data.Devices) (err error)
	DelDevices(devices data.Devices) (err error)
	GetDevices(devices data.Devices) (err error, deviceList []data.Devices)
	GetAllDevices() (err error, deviceList []data.Devices)
}

type DevicesServiceImpl struct {
	devices repo.DevicesRepo
}

func NewDevicesServiceImpl(devices repo.DevicesRepo) *DevicesServiceImpl {
	return &DevicesServiceImpl{devices: devices}
}

func (d DevicesServiceImpl) AddDevices(devices data.Devices) (err error) {
	return d.devices.Add(devices)
}

func (d DevicesServiceImpl) DelDevices(devices data.Devices) (err error) {
	return d.devices.Delete(devices)
}

func (d DevicesServiceImpl) GetAllDevices() (err error, deviceList []data.Devices) {
	err, deviceList = d.devices.FindAll()
	return
}

func (d DevicesServiceImpl) GetDevices(devices data.Devices) (err error, deviceList []data.Devices) {
	err, deviceList = d.devices.FindByStruct(devices)
	return
}
