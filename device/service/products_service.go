package service

import (
	"kv-iot/db"
	"kv-iot/device/data"
)

type products struct {
	db.BaseRepoI[data.Products]
}
