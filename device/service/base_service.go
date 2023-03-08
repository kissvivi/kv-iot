package service

// BaseService 建立依赖
type BaseService struct {
	ChannelsService   *ChannelsServiceImpl
	DevicesService    *DevicesServiceImpl
	KvActionService   *KvActionServiceImpl
	KvEventService    *KvEventServiceImpl
	KvPropertyService *KvPropertyServiceImpl
	ProductsService   *ProductsServiceImpl
}

func NewBaseService(channelsService *ChannelsServiceImpl, devicesService *DevicesServiceImpl, kvActionService *KvActionServiceImpl, kvEventService *KvEventServiceImpl, kvPropertyService *KvPropertyServiceImpl, productsService *ProductsServiceImpl) *BaseService {
	return &BaseService{ChannelsService: channelsService, DevicesService: devicesService, KvActionService: kvActionService, KvEventService: kvEventService, KvPropertyService: kvPropertyService, ProductsService: productsService}
}
