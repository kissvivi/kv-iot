package service

import (
	"kv-iot/db"
	"kv-iot/device/data"
)

type kvEventRepo struct {
	db.BaseRepoI[data.KvEvent]
}
