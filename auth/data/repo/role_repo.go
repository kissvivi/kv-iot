package repo

import "kv-iot/auth/data"

type RoleRepo struct {
	data.AuthRepo[data.Role]
}
