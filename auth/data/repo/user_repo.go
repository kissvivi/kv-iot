package repo

import "kv-iot/auth/data"

type UserRepo struct {
	data.AuthRepo[data.User]
}
