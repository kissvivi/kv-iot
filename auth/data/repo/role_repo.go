package repo

import (
	"kv-iot/auth/data"
	"kv-iot/db"

	"gorm.io/gorm"
)

type RoleRepo struct {
	data.AuthRepo[data.Role]
}

func (r RoleRepo) List() ([]data.Role, error) {
	roles := make([]data.Role, 0)
	err := db.MYSQLDB.Order("id ASC").Find(&roles).Error
	return roles, err
}

func (r RoleRepo) Get(id int64) (data.Role, error) {
	var role data.Role
	err := db.MYSQLDB.First(&role, id).Error
	return role, err
}

func (r RoleRepo) FindByName(name string) (data.Role, error) {
	var role data.Role
	err := db.MYSQLDB.Where("name = ?", name).First(&role).Error
	return role, err
}

func (r RoleRepo) Create(role *data.Role) error {
	return db.MYSQLDB.Create(role).Error
}

func (r RoleRepo) Update(id int64, fields map[string]interface{}) error {
	result := db.MYSQLDB.Model(&data.Role{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		exists, err := recordExists(db.MYSQLDB, &data.Role{}, id)
		if err != nil {
			return err
		}
		if !exists {
			return gorm.ErrRecordNotFound
		}
	}
	return nil
}

func (r RoleRepo) DeleteWithRelations(id int64) error {
	return db.MYSQLDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", id).Delete(&data.Permission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", id).Delete(&data.UserRole{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&data.Role{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r RoleRepo) CountByIDs(ids []int64) (int64, error) {
	ids = uniqueInt64(ids)
	if len(ids) == 0 {
		return 0, nil
	}
	var count int64
	err := db.MYSQLDB.Model(&data.Role{}).Where("id IN ?", ids).Count(&count).Error
	return count, err
}

func (r RoleRepo) RoleIDsByUserID(userID int64) ([]int64, error) {
	roleIDs := make([]int64, 0)
	err := db.MYSQLDB.Model(&data.UserRole{}).
		Where("user_id = ?", userID).
		Order("role_id ASC").
		Pluck("role_id", &roleIDs).Error
	return roleIDs, err
}

func (r RoleRepo) RoleNamesByUserID(userID int64) ([]string, error) {
	names := make([]string, 0)
	err := db.MYSQLDB.Table("role").
		Select("role.name").
		Joins("JOIN user_role ON user_role.role_id = role.id").
		Where("user_role.user_id = ? AND role.enable = ? AND role.deleted_at IS NULL", userID, 1).
		Group("role.id, role.name").
		Order("role.id ASC").
		Pluck("role.name", &names).Error
	return names, err
}

func (r RoleRepo) PermissionCodesByUserID(userID int64) ([]string, error) {
	codes := make([]string, 0)
	err := db.MYSQLDB.Table("permission").
		Select("permission.code").
		Joins("JOIN user_role ON user_role.role_id = permission.role_id").
		Joins("JOIN role ON role.id = permission.role_id").
		Where("user_role.user_id = ? AND role.enable = ? AND permission.code <> '' AND role.deleted_at IS NULL AND permission.deleted_at IS NULL", userID, 1).
		Group("permission.code").
		Order("permission.code ASC").
		Pluck("permission.code", &codes).Error
	return codes, err
}
