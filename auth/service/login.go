package service

import (
	"errors"
	"kv-iot/auth/data"
	"kv-iot/auth/data/repo"
)

var (
	ErrLogin = errors.New("用户名或密码错误")
)

var _ baseService = (*BaseServiceImpl)(nil)

type baseService interface {
	RegUser(user data.User) (err error, token string)
	Login(user data.User) (err error, token string)
}

type BaseServiceImpl struct {
	userRepo repo.UserRepo
}

func NewBaseServiceImpl(userRepo repo.UserRepo) *BaseServiceImpl {
	return &BaseServiceImpl{userRepo: userRepo}
}

func (b BaseServiceImpl) Login(user data.User) (err error, token string) {
	var m = make(map[string]interface{}, 0)
	var reUser []data.User
	m["user_name"] = user.UserName
	m["password"] = user.Password
	err, reUser = b.userRepo.FindBy(m)
	if err == nil && len(reUser) > 0 {
		token, err = GenerateToken(user)
		return
	}
	return ErrLogin, token
}

func (b BaseServiceImpl) RegUser(user data.User) (err error, token string) {
	if err = b.userRepo.Add(user); err == nil {
		token, err = GenerateToken(user)
	}
	return
}
