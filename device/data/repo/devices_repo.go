package repo

import (
	"kv-iot/db"
	"kv-iot/device/data"
)

type DevicesRepo struct {
	db.BaseRepoI[data.Devices]
}
