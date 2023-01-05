package service

import (
	"errors"
	"kv-iot/auth/data"
	"kv-iot/auth/data/repo"
)

var (
	ErrLogin = errors.New("用户名或密码错误")
)

var _ userService = (*UserServiceImpl)(nil)

type userService interface {
	RegUser(user data.User) (err error, token string)
	Login(user data.User) (err error, token string)
}

type UserServiceImpl struct {
	userRepo repo.UserRepo
}

func NewUserServiceImpl(userRepo repo.UserRepo) *UserServiceImpl {
	return &UserServiceImpl{userRepo: userRepo}
}

func (b UserServiceImpl) Login(user data.User) (err error, token string) {
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

func (b UserServiceImpl) RegUser(user data.User) (err error, token string) {
	if err = b.userRepo.Add(user); err == nil {
		token, err = GenerateToken(user)
	}
	return
}
