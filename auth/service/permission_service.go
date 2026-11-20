package service

import (
	"errors"
	"kv-iot/auth/data"
	"kv-iot/auth/data/repo"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrPermissionNotFound = errors.New("权限不存在")
	ErrPermissionExists   = errors.New("角色已拥有该模块权限")
)

type PermissionService interface {
	ListPermissions(roleID, moduleID int64) ([]data.Permission, error)
	GetPermission(id int64) (data.Permission, error)
	CreatePermission(permission data.Permission) (data.Permission, error)
	UpdatePermission(id int64, fields map[string]interface{}) (data.Permission, error)
	DeletePermission(id int64) error
	DeleteRoleModulePermission(roleID, moduleID int64) error
}

type PermissionServiceImpl struct {
	permissionRepo repo.PermissionRepo
	roleRepo       repo.RoleRepo
	moduleRepo     repo.ModuleRepo
}

func NewPermissionServiceImpl(permissionRepo repo.PermissionRepo, roleRepo repo.RoleRepo, moduleRepo repo.ModuleRepo) *PermissionServiceImpl {
	return &PermissionServiceImpl{
		permissionRepo: permissionRepo,
		roleRepo:       roleRepo,
		moduleRepo:     moduleRepo,
	}
}

func (s PermissionServiceImpl) ListPermissions(roleID, moduleID int64) ([]data.Permission, error) {
	return s.permissionRepo.List(roleID, moduleID)
}

func (s PermissionServiceImpl) GetPermission(id int64) (data.Permission, error) {
	permission, err := s.permissionRepo.Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return data.Permission{}, ErrPermissionNotFound
	}
	return permission, err
}

func (s PermissionServiceImpl) CreatePermission(permission data.Permission) (data.Permission, error) {
	if _, err := s.roleRepo.Get(permission.RoleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return data.Permission{}, ErrRoleNotFound
		}
		return data.Permission{}, err
	}
	module, err := s.moduleRepo.Get(permission.ModuleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return data.Permission{}, ErrModuleNotFound
		}
		return data.Permission{}, err
	}
	if _, err = s.permissionRepo.FindByRoleAndModule(permission.RoleID, permission.ModuleID); err == nil {
		return data.Permission{}, ErrPermissionExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return data.Permission{}, err
	}
	permission.Code = strings.TrimSpace(permission.Code)
	if permission.Code == "" {
		permission.Code = module.Code
	}
	if err = s.permissionRepo.Create(&permission); err != nil {
		return data.Permission{}, err
	}
	return permission, nil
}

func (s PermissionServiceImpl) UpdatePermission(id int64, fields map[string]interface{}) (data.Permission, error) {
	current, err := s.GetPermission(id)
	if err != nil {
		return data.Permission{}, err
	}
	targetRoleID := current.RoleID
	targetModuleID := current.ModuleID
	if roleValue, ok := fields["role_id"]; ok {
		roleID, valid := asInt64(roleValue)
		if !valid || roleID <= 0 {
			return data.Permission{}, ErrRoleNotFound
		}
		if _, err = s.roleRepo.Get(roleID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return data.Permission{}, ErrRoleNotFound
			}
			return data.Permission{}, err
		}
		targetRoleID = roleID
	}
	if moduleValue, ok := fields["module_id"]; ok {
		moduleID, valid := asInt64(moduleValue)
		if !valid || moduleID <= 0 {
			return data.Permission{}, ErrModuleNotFound
		}
		if _, err = s.moduleRepo.Get(moduleID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return data.Permission{}, ErrModuleNotFound
			}
			return data.Permission{}, err
		}
		targetModuleID = moduleID
	}
	if codeValue, ok := fields["code"]; ok {
		code, valid := codeValue.(string)
		if !valid {
			return data.Permission{}, ErrModuleCodeRequired
		}
		fields["code"] = strings.TrimSpace(code)
	}
	if targetRoleID != current.RoleID || targetModuleID != current.ModuleID {
		existing, findErr := s.permissionRepo.FindByRoleAndModule(targetRoleID, targetModuleID)
		if findErr == nil && existing.ID != current.ID {
			return data.Permission{}, ErrPermissionExists
		}
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return data.Permission{}, findErr
		}
	}
	if len(fields) > 0 {
		if err = s.permissionRepo.Update(id, fields); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return data.Permission{}, ErrPermissionNotFound
			}
			return data.Permission{}, err
		}
	}
	return s.GetPermission(id)
}

func asInt64(value interface{}) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case uint:
		return int64(typed), true
	case uint32:
		return int64(typed), true
	case uint64:
		return int64(typed), true
	default:
		return 0, false
	}
}

func (s PermissionServiceImpl) DeletePermission(id int64) error {
	err := s.permissionRepo.Delete(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrPermissionNotFound
	}
	return err
}

func (s PermissionServiceImpl) DeleteRoleModulePermission(roleID, moduleID int64) error {
	err := s.permissionRepo.DeleteByRoleAndModule(roleID, moduleID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrPermissionNotFound
	}
	return err
}
