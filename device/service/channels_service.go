package service

import (
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
)

var _ channelsService = (*ChannelsServiceImpl)(nil)

type channelsService interface {
	AddChannels(channels data.Channels) (err error)
	DelChannels(channels data.Channels) (err error)
	GetChannels(channels data.Channels) (err error, channelsList []data.Channels)
	GetAllChannels() (err error, channelsList []data.Channels)
}

type ChannelsServiceImpl struct {
	channels repo.ChannelsRepo
}

func NewChannelsServiceImpl(channels repo.ChannelsRepo) *ChannelsServiceImpl {
	return &ChannelsServiceImpl{channels: channels}
}

func (c ChannelsServiceImpl) AddChannels(channels data.Channels) (err error) {
	return c.channels.Add(channels)
}

func (c ChannelsServiceImpl) DelChannels(channels data.Channels) (err error) {
	return c.channels.Delete(channels)
}

func (c ChannelsServiceImpl) GetAllChannels() (err error, channelsList []data.Channels) {
	err, channelsList = c.channels.FindAll()
	return
}

func (c ChannelsServiceImpl) GetChannels(channels data.Channels) (err error, channelsList []data.Channels) {
	err, channelsList = c.channels.FindByStruct(channels)
	return
}
