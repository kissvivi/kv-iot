package repo

import (
	"kv-iot/auth/data"
)

type ModuleRepo struct {
	data.AuthRepo[data.User]
}
