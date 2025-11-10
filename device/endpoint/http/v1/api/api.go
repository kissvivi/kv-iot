package api

import (
	"kv-iot/device/endpoint/http/v1/api/channel"
	"kv-iot/device/endpoint/http/v1/api/device"
	"kv-iot/device/endpoint/http/v1/api/product"
)

// BaseApi 建立API依赖
type BaseApi struct {
	ApiDevice        *device.ApiDevice
	ApiDeviceConn    *device.ApiDeviceConn
	ApiProduct       *product.ApiProduct
	ApiProductModel  *product.ApiProductModel
	ApiChannelInstance *channel.ApiChannelInstance
}

// NewBaseApi 创建基础API实例
func NewBaseApi(apiDevice *device.ApiDevice, apiProduct *product.ApiProduct) *BaseApi {
	return &BaseApi{
		ApiDevice:  apiDevice,
		ApiProduct: apiProduct,
	}
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