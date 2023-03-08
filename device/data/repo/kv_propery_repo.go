package repo

import (
	"kv-iot/db"
	"kv-iot/device/data"
)

type KvPropertyRepo struct {
	db.BaseRepo[data.KvProperty]
}
