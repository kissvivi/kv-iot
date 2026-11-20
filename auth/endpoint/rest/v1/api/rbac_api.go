package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"kv-iot/auth/data"
	"kv-iot/auth/service"
)

type RoleApiImpl struct {
	baseService service.BaseService
}

type ModuleApiImpl struct {
	baseService service.BaseService
}

type PermissionApiImpl struct {
	baseService service.BaseService
}

type createRoleRequest struct {
	Name   string `json:"name"`
	Desc   string `json:"desc"`
	Enable *int   `json:"enable"`
}

type updateRoleRequest struct {
	Name   *string `json:"name"`
	Desc   *string `json:"desc"`
	Enable *int    `json:"enable"`
}

type moduleRequest struct {
	Name       string `json:"name"`
	Uri        string `json:"uri"`
	Code       string `json:"code"`
	ParentID   *int64 `json:"parent_id"`
	ParentCode *int64 `json:"parent_code"`
	Enable     *int   `json:"enable"`
}

type updateModuleRequest struct {
	Name       *string `json:"name"`
	Uri        *string `json:"uri"`
	Code       *string `json:"code"`
	ParentID   *int64  `json:"parent_id"`
	ParentCode *int64  `json:"parent_code"`
	Enable     *int    `json:"enable"`
}

type permissionRequest struct {
	RoleID   int64  `json:"role_id"`
	ModuleID int64  `json:"module_id"`
	Code     string `json:"code"`
}

type updatePermissionRequest struct {
	RoleID   *int64  `json:"role_id"`
	ModuleID *int64  `json:"module_id"`
	Code     *string `json:"code"`
}

type assignPermissionsRequest struct {
	ModuleIDs []int64 `json:"module_ids"`
}

func NewRoleApiImpl(baseService *service.BaseService) *RoleApiImpl {
	return &RoleApiImpl{baseService: *baseService}
}

func NewModuleApiImpl(baseService *service.BaseService) *ModuleApiImpl {
	return &ModuleApiImpl{baseService: *baseService}
}

func NewPermissionApiImpl(baseService *service.BaseService) *PermissionApiImpl {
	return &PermissionApiImpl{baseService: *baseService}
}

func (a RoleApiImpl) ListRoles(c *gin.Context) {
	roles, err := a.baseService.RoleService.ListRoles()
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, roles, "查询成功")
}

func (a RoleApiImpl) GetRole(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	role, err := a.baseService.RoleService.GetRole(id)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, role, "查询成功")
}

func (a RoleApiImpl) CreateRole(c *gin.Context) {
	var request createRoleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}
	enable := 1
	if request.Enable != nil {
		enable = *request.Enable
	}
	role, err := a.baseService.RoleService.CreateRole(data.Role{
		Name:   request.Name,
		Desc:   request.Desc,
		Enable: enable,
	})
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, role, "创建成功")
}

func (a RoleApiImpl) UpdateRole(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request updateRoleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}
	fields := make(map[string]interface{})
	if request.Name != nil {
		fields["name"] = *request.Name
	}
	if request.Desc != nil {
		fields["desc"] = *request.Desc
	}
	if request.Enable != nil {
		fields["enable"] = *request.Enable
	}
	role, err := a.baseService.RoleService.UpdateRole(id, fields)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, role, "更新成功")
}

func (a RoleApiImpl) DeleteRole(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := a.baseService.RoleService.DeleteRole(id); err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, nil, "删除成功")
}

func (a RoleApiImpl) GetRolePermissions(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	permissions, err := a.baseService.RoleService.GetRolePermissions(id)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, permissions, "查询成功")
}

func (a RoleApiImpl) AssignRolePermissions(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request assignPermissionsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}
	if err := a.baseService.RoleService.AssignPermissions(id, request.ModuleIDs); err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	permissions, err := a.baseService.RoleService.GetRolePermissions(id)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, permissions, "权限分配成功")
}

func (a ModuleApiImpl) ListModules(c *gin.Context) {
	modules, err := a.baseService.ModuleService.ListModules()
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, modules, "查询成功")
}

func (a ModuleApiImpl) GetModule(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	module, err := a.baseService.ModuleService.GetModule(id)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, module, "查询成功")
}

func (a ModuleApiImpl) CreateModule(c *gin.Context) {
	var request moduleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}
	module, err := a.baseService.ModuleService.CreateModule(moduleFromRequest(request))
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, module, "创建成功")
}

func (a ModuleApiImpl) UpdateModule(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request updateModuleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}
	fields := make(map[string]interface{})
	if request.Name != nil {
		fields["name"] = *request.Name
	}
	if request.Uri != nil {
		fields["uri"] = *request.Uri
	}
	if request.Code != nil {
		fields["code"] = *request.Code
	}
	if request.ParentID != nil {
		fields["parent_id"] = *request.ParentID
	}
	if request.ParentCode != nil {
		fields["parent_code"] = *request.ParentCode
	}
	if request.Enable != nil {
		fields["enable"] = *request.Enable
	}
	module, err := a.baseService.ModuleService.UpdateModule(id, fields)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, module, "更新成功")
}

func (a ModuleApiImpl) DeleteModule(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := a.baseService.ModuleService.DeleteModule(id); err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, nil, "删除成功")
}

func (a PermissionApiImpl) ListPermissions(c *gin.Context) {
	roleID, _ := strconv.ParseInt(c.Query("role_id"), 10, 64)
	moduleID, _ := strconv.ParseInt(c.Query("module_id"), 10, 64)
	permissions, err := a.baseService.PermissionService.ListPermissions(roleID, moduleID)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, permissions, "查询成功")
}

func (a PermissionApiImpl) GetPermission(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	permission, err := a.baseService.PermissionService.GetPermission(id)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, permission, "查询成功")
}

func (a PermissionApiImpl) CreatePermission(c *gin.Context) {
	var request permissionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}
	permission, err := a.baseService.PermissionService.CreatePermission(data.Permission{
		RoleID:   request.RoleID,
		ModuleID: request.ModuleID,
		Code:     request.Code,
	})
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, permission, "创建成功")
}

func (a PermissionApiImpl) UpdatePermission(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request updatePermissionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}
	fields := make(map[string]interface{})
	if request.RoleID != nil {
		fields["role_id"] = *request.RoleID
	}
	if request.ModuleID != nil {
		fields["module_id"] = *request.ModuleID
	}
	if request.Code != nil {
		fields["code"] = *request.Code
	}
	permission, err := a.baseService.PermissionService.UpdatePermission(id, fields)
	if err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, permission, "更新成功")
}

func (a PermissionApiImpl) DeletePermission(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := a.baseService.PermissionService.DeletePermission(id); err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, nil, "删除成功")
}

func (a PermissionApiImpl) DeleteRoleModulePermission(c *gin.Context) {
	roleID, roleErr := strconv.ParseInt(c.Query("role_id"), 10, 64)
	moduleID, moduleErr := strconv.ParseInt(c.Query("module_id"), 10, 64)
	if roleErr != nil || moduleErr != nil || roleID <= 0 || moduleID <= 0 {
		fail(c, http.StatusBadRequest, "role_id和module_id不能为空")
		return
	}
	if err := a.baseService.PermissionService.DeleteRoleModulePermission(roleID, moduleID); err != nil {
		fail(c, http.StatusOK, err.Error())
		return
	}
	success(c, nil, "删除成功")
}

func moduleFromRequest(request moduleRequest) data.Module {
	enable := 1
	if request.Enable != nil {
		enable = *request.Enable
	}
	module := data.Module{
		Name:   request.Name,
		Uri:    request.Uri,
		Code:   request.Code,
		Enable: enable,
	}
	if request.ParentID != nil {
		module.ParentID = *request.ParentID
	}
	if request.ParentCode != nil {
		module.ParentCode = *request.ParentCode
	}
	return module
}
