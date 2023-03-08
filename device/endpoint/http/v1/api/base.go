package api

import (
	"kv-iot/device/endpoint/http/v1/api/baseapi"
)

type BaseApi struct {
	DeviceApi *baseapi.DeviceApi
}

func NewBaseApi(deviceApi *baseapi.DeviceApi) *BaseApi {
	return &BaseApi{DeviceApi: deviceApi}
}
