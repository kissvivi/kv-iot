package channels

import (
	"github.com/gin-gonic/gin"
	"kv-iot/device/data"
	"kv-iot/device/service"
	"kv-iot/pkg/result"
)

type ApiChannels struct {
	baseService *service.BaseService
}

func NewApiChannels(channelsService *service.BaseService) *ApiChannels {
	return &ApiChannels{baseService: channelsService}
}

func (ac ApiChannels) AddChannels(c *gin.Context) {
	channels := data.Channels{}
	if err := c.BindJSON(&channels); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误")
		return
	}

	err := ac.baseService.ChannelsService.AddChannels(channels)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "添加失败")
	} else {
		result.BaseResult{}.SuccessResult(c, channels, "添加成功")
	}
}

func (ac ApiChannels) DelChannels(c *gin.Context) {
	channels := data.Channels{}
	if err := c.BindJSON(&channels); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误")
		return
	}

	err := ac.baseService.ChannelsService.DelChannels(channels)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "删除失败")
	} else {
		result.BaseResult{}.SuccessResult(c, nil, "删除成功")
	}
}

func (ac ApiChannels) GetChannels(c *gin.Context) {
	channels := data.Channels{}
	if err := c.BindJSON(&channels); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误")
		return
	}

	err, channelsList := ac.baseService.ChannelsService.GetChannels(channels)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败")
	} else {
		result.BaseResult{}.SuccessResult(c, channelsList, "查询成功")
	}
}

func (ac ApiChannels) GetAllChannels(c *gin.Context) {
	err, channelsList := ac.baseService.ChannelsService.GetAllChannels()
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败")
	} else {
		result.BaseResult{}.SuccessResult(c, channelsList, "查询成功")
	}
}
