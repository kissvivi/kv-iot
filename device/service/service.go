package service

import (
	"fmt"
	"github.kissvivi.kv-iot/device/data"
)

var _ baseService = (*BaseServiceImpl)(nil)

type baseService interface {
	AddDevice(devices data.Devices)
}

type BaseServiceImpl struct {
	deviceRepo data.DevicesRepo
}

func (b BaseServiceImpl) AddDevice(devices data.Devices) {
	fmt.Println(devices)
	b.deviceRepo.Add(devices)
}
