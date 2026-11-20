package service

// BaseService is the service container used by the auth HTTP adapter.
type BaseService struct {
	UserService       UserService
	RoleService       RoleService
	ModuleService     ModuleService
	PermissionService PermissionService
}

func NewBaseService(
	userService UserService,
	roleService RoleService,
	moduleService ModuleService,
	permissionService PermissionService,
) *BaseService {
	return &BaseService{
		UserService:       userService,
		RoleService:       roleService,
		ModuleService:     moduleService,
		PermissionService: permissionService,
	}
}
