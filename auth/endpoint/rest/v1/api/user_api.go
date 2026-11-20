package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"kv-iot/auth/data"
	"kv-iot/auth/service"
)

var _ userApi = (*UserApiImpl)(nil)

type userApi interface {
	RegUser(c *gin.Context)
	Login(c *gin.Context)
	ListUsers(c *gin.Context)
	GetUser(c *gin.Context)
	CreateUser(c *gin.Context)
	UpdateUser(c *gin.Context)
	DeleteUser(c *gin.Context)
	Me(c *gin.Context)
}

type UserApiImpl struct {
	userBaseService service.BaseService
}

type loginRequest struct {
	UserName      string `json:"userName"`
	UserNameSnake string `json:"user_name"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

func (r loginRequest) username() string {
	if strings.TrimSpace(r.UserName) != "" {
		return strings.TrimSpace(r.UserName)
	}
	if strings.TrimSpace(r.UserNameSnake) != "" {
		return strings.TrimSpace(r.UserNameSnake)
	}
	return strings.TrimSpace(r.Username)
}

type createUserRequest struct {
	UserName      string  `json:"userName"`
	UserNameSnake string  `json:"user_name"`
	Password      string  `json:"password"`
	IsAdmin       *bool   `json:"isAdmin"`
	IsAdminSnake  *bool   `json:"is_admin_flag"`
	RoleIDs       []int64 `json:"roleIds"`
	RoleIDsSnake  []int64 `json:"role_ids"`
}

func (r createUserRequest) username() string {
	if strings.TrimSpace(r.UserName) != "" {
		return strings.TrimSpace(r.UserName)
	}
	return strings.TrimSpace(r.UserNameSnake)
}

func (r createUserRequest) isAdmin() bool {
	if r.IsAdmin != nil {
		return *r.IsAdmin
	}
	if r.IsAdminSnake != nil {
		return *r.IsAdminSnake
	}
	return false
}

func (r createUserRequest) roleIDs() []int64 {
	if r.RoleIDs != nil {
		return r.RoleIDs
	}
	return r.RoleIDsSnake
}

type updateUserRequest struct {
	UserName      *string  `json:"userName"`
	UserNameSnake *string  `json:"user_name"`
	Password      *string  `json:"password"`
	IsAdmin       *bool    `json:"isAdmin"`
	IsAdminSnake  *bool    `json:"is_admin_flag"`
	RoleIDs       *[]int64 `json:"roleIds"`
	RoleIDsSnake  *[]int64 `json:"role_ids"`
}

func NewUserApiImpl(userBaseService *service.BaseService) *UserApiImpl {
	return &UserApiImpl{userBaseService: *userBaseService}
}

func (u UserApiImpl) RegUser(c *gin.Context) {
	var request createUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}
	user := data.User{
		UserName: request.username(),
		Password: request.Password,
	}
	err, token := u.userBaseService.UserService.RegUser(user)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, token, "注册成功")
}

func (u UserApiImpl) Login(c *gin.Context) {
	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}
	user := data.User{
		UserName: request.username(),
		Password: request.Password,
	}
	err, token := u.userBaseService.UserService.Login(user)
	if err != nil {
		fail(c, http.StatusOK, "用户名或密码错误")
		return
	}
	success(c, token, "登录成功")
}

func (u UserApiImpl) ListUsers(c *gin.Context) {
	users, err := u.userBaseService.UserService.ListUsers()
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, users, "查询成功")
}

func (u UserApiImpl) GetUser(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user, err := u.userBaseService.UserService.GetUser(id)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, user, "查询成功")
}

func (u UserApiImpl) CreateUser(c *gin.Context) {
	var request createUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}
	user, err := u.userBaseService.UserService.CreateUser(service.CreateUserInput{
		UserName: request.username(),
		Password: request.Password,
		IsAdmin:  request.isAdmin(),
		RoleIDs:  request.roleIDs(),
	})
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, user, "创建成功")
}

func (u UserApiImpl) UpdateUser(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request updateUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}
	userName := request.UserName
	if userName == nil {
		userName = request.UserNameSnake
	}
	isAdmin := request.IsAdmin
	if isAdmin == nil {
		isAdmin = request.IsAdminSnake
	}
	roleIDs := request.RoleIDs
	if roleIDs == nil {
		roleIDs = request.RoleIDsSnake
	}
	user, err := u.userBaseService.UserService.UpdateUser(id, service.UpdateUserInput{
		UserName: userName,
		Password: request.Password,
		IsAdmin:  isAdmin,
		RoleIDs:  roleIDs,
	})
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, user, "更新成功")
}

func (u UserApiImpl) DeleteUser(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := u.userBaseService.UserService.DeleteUser(id); err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, nil, "删除成功")
}

func (u UserApiImpl) Me(c *gin.Context) {
	claims, ok := service.ClaimsFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "认证信息缺失")
		return
	}
	success(c, claims, "查询成功")
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		fail(c, http.StatusBadRequest, "资源ID格式错误")
		return 0, false
	}
	return id, true
}
