package service

import (
	"kv-iot/db"
	"kv-iot/device/data/repo"
	"sync"
)

// BaseService 建立依赖
type BaseService struct {
	ChannelsService               *ChannelsServiceImpl
	DevicesService                *DevicesServiceImpl
	DeviceConnService             deviceConnService
	KvActionService               *KvActionServiceImpl
	KvEventService                *KvEventServiceImpl
	KvPropertyService             *KvPropertyServiceImpl
	ProductsService               *ProductsServiceImpl
	ProductModelService           ProductModelService
	ChannelInstanceService        ChannelInstanceService
	ProductProtocolProfileService ProductProtocolProfileService
	ProductPointBindingService    ProductPointBindingService
	DevicePointBindingService     DevicePointBindingService
	mu                            sync.Mutex
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

func (s *BaseService) GetProductProtocolProfileService() ProductProtocolProfileService {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ProductProtocolProfileService == nil {
		s.ProductProtocolProfileService = NewProductProtocolProfileService(
			repo.NewProductProtocolProfileRepo(db.MYSQLDB),
			s.ProductsService,
		)
	}
	return s.ProductProtocolProfileService
}

func (s *BaseService) GetProductPointBindingService() ProductPointBindingService {
	profiles := s.GetProductProtocolProfileService()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ProductModelService == nil {
		s.ProductModelService = NewProductModelServiceImpl(s.KvPropertyService, s.KvActionService, s.KvEventService)
	}
	if s.ProductPointBindingService == nil {
		s.ProductPointBindingService = NewProductPointBindingService(
			repo.NewProductPointBindingRepo(db.MYSQLDB),
			profiles,
			s.ProductModelService,
		)
	}
	return s.ProductPointBindingService
}

func (s *BaseService) GetDevicePointBindingService() DevicePointBindingService {
	productBindings := s.GetProductPointBindingService()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.DevicePointBindingService == nil {
		s.DevicePointBindingService = NewDevicePointBindingService(
			repo.NewDevicePointBindingRepo(db.MYSQLDB),
			s.DevicesService,
			productBindings,
			s.ChannelInstanceService,
		)
	}
	return s.DevicePointBindingService
}
