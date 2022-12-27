package service

import "kv-iot/auth/data"

var _ baseService = (*BaseServiceImpl)(nil)

type baseService interface {
	RegUser(user data.User)
}

type BaseServiceImpl struct {
	authRepo data.AuthRepo
}

func (b BaseServiceImpl) RegUser(user data.User) {
	b.authRepo.Add(user)
}
