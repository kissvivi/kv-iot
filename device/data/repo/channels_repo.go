package repo

import (
	"kv-iot/db"
	"kv-iot/device/data"
)

type channelsRepo struct {
	db.BaseRepoI[data.Channels]
}
