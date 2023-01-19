package repo

import (
	"kv-iot/db"
	"kv-iot/device/data"
)

type KvProperty struct {
	db.BaseRepoI[data.KvProperty]
}
