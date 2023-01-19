package api

import (
	"github.com/gin-gonic/gin"
	"kv-iot/auth/data"
	"kv-iot/auth/service"
	"kv-iot/pkg/result"
)

var _ userApi = (*UserApiImpl)(nil)

type userApi interface {
	RegUser(c *gin.Context)
	Login(c *gin.Context)
}

type UserApiImpl struct {
	userBaseService service.BaseService
}

func NewUserApiImpl(userBaseService *service.BaseService) *UserApiImpl {
	return &UserApiImpl{userBaseService: *userBaseService}
}

func (u UserApiImpl) RegUser(c *gin.Context) {
	user := data.User{}
	c.BindJSON(&user)

	err, token := u.userBaseService.UserService.RegUser(user)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, err.Error())
	}
	result.BaseResult{}.SuccessResult(c, token, "注册成功")
}

func (u UserApiImpl) Login(c *gin.Context) {
	user := data.User{}
	c.ShouldBind(&user)

	err, token := u.userBaseService.UserService.Login(user)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "用户名或密码错误")
	} else {
		result.BaseResult{}.SuccessResult(c, token, "登录成功")
	}

}
