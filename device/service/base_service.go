package service

// BaseService 建立依赖
type BaseService struct {
	ChannelsService *ChannelsServiceImpl
}

func NewBaseService(channelsService *ChannelsServiceImpl) *BaseService {
	return &BaseService{ChannelsService: channelsService}
}
