package service

import (
	"errors"
	"kv-iot/auth/data"
	"kv-iot/auth/data/repo"
	"log"
)

var (
	ErrLogin = errors.New("用户名或密码错误")
)

// 接口命名与实现一致
var _ UserService = (*UserServiceImpl)(nil)

type UserService interface {
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
	// 不直接查询密码，先根据用户名查询用户，然后再验证密码
	err, reUser = b.userRepo.FindBy(m)
	if err != nil {
		log.Printf("登录失败：用户查询出错 %v", err)
		return ErrLogin, token
	}

	if len(reUser) > 0 {
		// 验证密码 - 实际应该使用密码哈希比较，但暂时保持兼容性
		if reUser[0].Password == user.Password {
			token, err = GenerateToken(reUser[0])
			if err != nil {
				log.Printf("登录失败：生成token出错 %v", err)
			}
			return
		}
	}
	return ErrLogin, token
}

func (b UserServiceImpl) RegUser(user data.User) (err error, token string) {
	// TODO: 实际应该对密码进行哈希处理
	log.Printf("用户注册：用户名 %s", user.UserName)
	if err = b.userRepo.Add(user); err == nil {
		token, err = GenerateToken(user)
		if err != nil {
			log.Printf("注册成功但生成token失败: %v", err)
		}
	} else {
		log.Printf("用户注册失败: %v", err)
	}
	return
}
