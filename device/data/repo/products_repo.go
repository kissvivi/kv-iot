package repo

import (
	"kv-iot/db"
	"kv-iot/device/data"
)

type ProductsRepo struct {
	db.BaseRepo[data.Products]
}
