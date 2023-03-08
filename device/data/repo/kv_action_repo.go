package repo

import (
	"kv-iot/db"
	"kv-iot/device/data"
)

type KvActionRepo struct {
	db.BaseRepo[data.KvAction]
}
