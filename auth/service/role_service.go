package service

import (
	"errors"
	"kv-iot/auth/data"
	"kv-iot/auth/data/repo"
	"log"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrRoleExists       = errors.New("角色名称已存在")
	ErrRoleNameRequired = errors.New("角色名称不能为空")
	ErrModuleNotFound   = errors.New("模块不存在")
)

var _ RoleService = (*RoleServiceImpl)(nil)

type RoleService interface {
	AddRole(role data.Role) (err error)
	DelRole(role data.Role) (err error)
	ListRoles() ([]data.Role, error)
	GetRole(id int64) (data.Role, error)
	CreateRole(role data.Role) (data.Role, error)
	UpdateRole(id int64, fields map[string]interface{}) (data.Role, error)
	DeleteRole(id int64) error
	GetRolePermissions(id int64) ([]data.Permission, error)
	AssignPermissions(roleID int64, moduleIDs []int64) error
}

type RoleServiceImpl struct {
	roleRepo       repo.RoleRepo
	moduleRepo     repo.ModuleRepo
	permissionRepo repo.PermissionRepo
}

func NewRoleServiceImpl(roleRepo repo.RoleRepo, moduleRepo repo.ModuleRepo, permissionRepo repo.PermissionRepo) *RoleServiceImpl {
	return &RoleServiceImpl{
		roleRepo:       roleRepo,
		moduleRepo:     moduleRepo,
		permissionRepo: permissionRepo,
	}
}

func (r RoleServiceImpl) AddRole(role data.Role) (err error) {
	_, err = r.CreateRole(role)
	return err
}

func (r RoleServiceImpl) DelRole(role data.Role) (err error) {
	return r.DeleteRole(int64(role.ID))
}

func (r RoleServiceImpl) ListRoles() ([]data.Role, error) {
	return r.roleRepo.List()
}

func (r RoleServiceImpl) GetRole(id int64) (data.Role, error) {
	role, err := r.roleRepo.Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return data.Role{}, ErrRoleNotFound
	}
	return role, err
}

func (r RoleServiceImpl) CreateRole(role data.Role) (data.Role, error) {
	role.Name = strings.TrimSpace(role.Name)
	if role.Name == "" {
		return data.Role{}, ErrRoleNameRequired
	}
	if _, err := r.roleRepo.FindByName(role.Name); err == nil {
		return data.Role{}, ErrRoleExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return data.Role{}, err
	}
	if err := r.roleRepo.Create(&role); err != nil {
		return data.Role{}, err
	}
	return role, nil
}

func (r RoleServiceImpl) UpdateRole(id int64, fields map[string]interface{}) (data.Role, error) {
	if name, ok := fields["name"].(string); ok {
		name = strings.TrimSpace(name)
		if name == "" {
			return data.Role{}, ErrRoleNameRequired
		}
		if existing, err := r.roleRepo.FindByName(name); err == nil && int64(existing.ID) != id {
			return data.Role{}, ErrRoleExists
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return data.Role{}, err
		}
		fields["name"] = name
	}
	if len(fields) > 0 {
		if err := r.roleRepo.Update(id, fields); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return data.Role{}, ErrRoleNotFound
			}
			return data.Role{}, err
		}
	}
	return r.GetRole(id)
}

func (r RoleServiceImpl) DeleteRole(id int64) error {
	err := r.roleRepo.DeleteWithRelations(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrRoleNotFound
	}
	return err
}

func (r RoleServiceImpl) GetRolePermissions(id int64) ([]data.Permission, error) {
	if _, err := r.GetRole(id); err != nil {
		return nil, err
	}
	return r.permissionRepo.ListByRoleID(id)
}

func (r RoleServiceImpl) AssignPermissions(roleID int64, moduleIDs []int64) error {
	if _, err := r.GetRole(roleID); err != nil {
		return err
	}
	normalized := uniquePositiveIDs(moduleIDs)
	modules, err := r.moduleRepo.FindByIDs(normalized)
	if err != nil {
		return err
	}
	if len(modules) != len(normalized) {
		return ErrModuleNotFound
	}

	permissions := make([]data.Permission, 0, len(modules))
	for _, module := range modules {
		permissions = append(permissions, data.Permission{
			RoleID:   roleID,
			ModuleID: int64(module.ID),
			Code:     module.Code,
		})
	}
	if err = r.permissionRepo.ReplaceRolePermissions(roleID, permissions); err != nil {
		log.Printf("分配角色权限失败，role_id=%d: %v", roleID, err)
		return err
	}
	return nil
}

func uniquePositiveIDs(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
