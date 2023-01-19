package service

import (
	"kv-iot/db"
	"kv-iot/device/data"
)

type kvActionRepo struct {
	db.BaseRepoI[data.KvAction]
}
