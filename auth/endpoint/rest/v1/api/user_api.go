package api

import (
	"github.com/gin-gonic/gin"
	"kv-iot/auth/data"
	"kv-iot/auth/endpoint/rest"
	"kv-iot/auth/service"
)

var _ userApi = (*UserApiImpl)(nil)

type userApi interface {
	RegUser(c *gin.Context)
	Login(c *gin.Context)
}

type UserApiImpl struct {
	userBaseService service.BaseServiceImpl
}

func NewUserApiImpl(userBaseService *service.BaseServiceImpl) *UserApiImpl {
	return &UserApiImpl{userBaseService: *userBaseService}
}

func (u UserApiImpl) RegUser(c *gin.Context) {
	user := data.User{}
	c.BindJSON(&user)

	err, token := u.userBaseService.RegUser(user)
	if err != nil {
		rest.BaseResult{}.ErrResult(c, nil, err.Error())
	}
	rest.BaseResult{}.SuccessResult(c, token, "注册成功")
}

func (u UserApiImpl) Login(c *gin.Context) {
	user := data.User{}
	c.ShouldBind(&user)

	err, token := u.userBaseService.Login(user)
	if err != nil {
		rest.BaseResult{}.ErrResult(c, nil, "用户名或密码错误")
	} else {
		rest.BaseResult{}.SuccessResult(c, token, "登录成功")
	}

}
