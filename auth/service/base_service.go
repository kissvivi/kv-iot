package service

// BaseService 服务容器，使用接口而非具体实现
// 这样可以更容易进行测试和替换具体实现
type BaseService struct {
	UserService UserService
	RoleService RoleService
}

// NewBaseService 创建服务容器
// 使用接口类型作为参数，提高代码灵活性和可测试性
func NewBaseService(userService UserService, roleService RoleService) *BaseService {
	return &BaseService{
		UserService: userService,
		RoleService: roleService,
	}
}
