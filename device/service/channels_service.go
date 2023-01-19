package service

import (
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
)

var _ channelsService = (*ChannelsServiceImpl)(nil)

type channelsService interface {
	AddChannels(channels data.Channels) (err error)
	DelChannels(channels data.Channels) (err error)
}

type ChannelsServiceImpl struct {
	channels repo.ChannelsRepo
}

func (c ChannelsServiceImpl) AddChannels(channels data.Channels) (err error) {
	//TODO implement me
	panic("implement me")
}

func (c ChannelsServiceImpl) DelChannels(channels data.Channels) (err error) {
	//TODO implement me
	panic("implement me")
}
