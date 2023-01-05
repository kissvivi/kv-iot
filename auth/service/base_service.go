package service

type BaseService struct {
	UserService *UserServiceImpl
	RoleService *RoleServiceImpl
}

func NewBaseService(userServiceImpl *UserServiceImpl, roleService *RoleServiceImpl) *BaseService {
	return &BaseService{UserService: userServiceImpl, RoleService: roleService}
}
