package api

import (
	"kv-iot/device/endpoint/http/v1/api/channel"
	"kv-iot/device/endpoint/http/v1/api/device"
	"kv-iot/device/endpoint/http/v1/api/product"
)

// BaseApi 建立API依赖
type BaseApi struct {
	ApiDevice                 *device.ApiDevice
	ApiDeviceConn             *device.ApiDeviceConn
	ApiDevicePointBinding     *device.ApiDevicePointBinding
	ApiProduct                *product.ApiProduct
	ApiProductModel           *product.ApiProductModel
	ApiProductProtocolProfile *product.ApiProductProtocolProfile
	ApiProductPointBinding    *product.ApiProductPointBinding
	ApiChannelInstance        *channel.ApiChannelInstance
}

// NewBaseApi 创建基础API实例
func NewBaseApi(apiDevice *device.ApiDevice, apiProduct *product.ApiProduct) *BaseApi {
	baseApi := &BaseApi{
		ApiDevice:  apiDevice,
		ApiProduct: apiProduct,
	}
	if apiDevice != nil {
		baseApi.ApiDevicePointBinding = device.NewApiDevicePointBinding(apiDevice.BaseService())
	}
	if apiProduct != nil {
		baseApi.ApiProductProtocolProfile = product.NewApiProductProtocolProfile(apiProduct.BaseService())
		baseApi.ApiProductPointBinding = product.NewApiProductPointBinding(apiProduct.BaseService())
	}
	return baseApi
}

// SetApiDeviceConn 设置设备连接API
func (a *BaseApi) SetApiDeviceConn(apiDeviceConn *device.ApiDeviceConn) {
	a.ApiDeviceConn = apiDeviceConn
}

// SetApiProductModel 设置产品物模型API
func (a *BaseApi) SetApiProductModel(apiProductModel *product.ApiProductModel) {
	a.ApiProductModel = apiProductModel
}

// SetApiChannelInstance 设置通道实例API
func (a *BaseApi) SetApiChannelInstance(apiChannelInstance *channel.ApiChannelInstance) {
	a.ApiChannelInstance = apiChannelInstance
}
