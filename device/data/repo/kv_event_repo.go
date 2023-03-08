package repo

import (
	"kv-iot/db"
	"kv-iot/device/data"
)

type KvEventRepo struct {
	db.BaseRepo[data.KvEvent]
}
