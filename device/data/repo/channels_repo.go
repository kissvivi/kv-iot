package repo

import (
	"kv-iot/db"
	"kv-iot/device/data"
)

type ChannelsRepo struct {
	db.BaseRepo[data.Channels]
}
