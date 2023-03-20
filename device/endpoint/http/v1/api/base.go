package api

import (
	"kv-iot/device/endpoint/http/v1/api/device"
	"kv-iot/device/endpoint/http/v1/api/product"
)

type BaseApi struct {
	ApiDevice  *device.ApiDevice
	ApiProduct *product.ApiProduct
}

func NewBaseApi(apiDevice *device.ApiDevice, apiProduct *product.ApiProduct) *BaseApi {
	return &BaseApi{ApiDevice: apiDevice, ApiProduct: apiProduct}
}
