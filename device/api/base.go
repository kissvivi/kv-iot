package api

import "kv-iot/device/api/baseapi"

type BaseApi struct {
	DeviceApi baseapi.DeviceApi
}

func NewBaseApi() *BaseApi {
	return &BaseApi{}
}
