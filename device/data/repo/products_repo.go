package repo

import (
	"kv-iot/db"
	"kv-iot/device/data"
)

type Products struct {
	db.BaseRepoI[data.Products]
}
