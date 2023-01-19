package service

import (
	"kv-iot/db"
	"kv-iot/device/data"
)

type kvProperty struct {
	db.BaseRepoI[data.KvProperty]
}
