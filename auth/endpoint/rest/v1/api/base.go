package api

type BaseApi struct {
	UserApi       UserApiImpl
	RoleApi       RoleApiImpl
	ModuleApi     ModuleApiImpl
	PermissionApi PermissionApiImpl
}

func NewBaseApi(
	userApi *UserApiImpl,
	roleApi *RoleApiImpl,
	moduleApi *ModuleApiImpl,
	permissionApi *PermissionApiImpl,
) *BaseApi {
	return &BaseApi{
		UserApi:       *userApi,
		RoleApi:       *roleApi,
		ModuleApi:     *moduleApi,
		PermissionApi: *permissionApi,
	}
}
