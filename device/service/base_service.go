package service

// BaseService 建立依赖
type BaseService struct {
	ChannelsService    *ChannelsServiceImpl
	DevicesService     *DevicesServiceImpl
	DeviceConnService  deviceConnService
	KvActionService    *KvActionServiceImpl
	KvEventService     *KvEventServiceImpl
	KvPropertyService  *KvPropertyServiceImpl
	ProductsService    *ProductsServiceImpl
	ProductModelService ProductModelService
	ChannelInstanceService ChannelInstanceService
}

func NewBaseService(channelsService *ChannelsServiceImpl, devicesService *DevicesServiceImpl, kvActionService *KvActionServiceImpl, kvEventService *KvEventServiceImpl, kvPropertyService *KvPropertyServiceImpl, productsService *ProductsServiceImpl) *BaseService {
	return &BaseService{ChannelsService: channelsService, DevicesService: devicesService, KvActionService: kvActionService, KvEventService: kvEventService, KvPropertyService: kvPropertyService, ProductsService: productsService}
}

// SetDeviceConnService 设置设备连接服务
func (s *BaseService) SetDeviceConnService(deviceConnService deviceConnService) {
	s.DeviceConnService = deviceConnService
}

// SetProductModelService 设置产品物模型服务
func (s *BaseService) SetProductModelService(productModelService ProductModelService) {
	s.ProductModelService = productModelService
}

// SetChannelInstanceService 设置通道实例服务
func (s *BaseService) SetChannelInstanceService(channelInstanceService ChannelInstanceService) {
	s.ChannelInstanceService = channelInstanceService
}
