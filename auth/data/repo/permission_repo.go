package repo

import (
	"kv-iot/auth/data"
	"kv-iot/db"

	"gorm.io/gorm"
)

type PermissionRepo struct {
	data.AuthRepo[data.Permission]
}

func (r PermissionRepo) List(roleID, moduleID int64) ([]data.Permission, error) {
	permissions := make([]data.Permission, 0)
	query := db.MYSQLDB.Order("id ASC")
	if roleID > 0 {
		query = query.Where("role_id = ?", roleID)
	}
	if moduleID > 0 {
		query = query.Where("module_id = ?", moduleID)
	}
	err := query.Find(&permissions).Error
	return permissions, err
}

func (r PermissionRepo) Get(id int64) (data.Permission, error) {
	var permission data.Permission
	err := db.MYSQLDB.First(&permission, id).Error
	return permission, err
}

func (r PermissionRepo) FindByRoleAndModule(roleID, moduleID int64) (data.Permission, error) {
	var permission data.Permission
	err := db.MYSQLDB.Where("role_id = ? AND module_id = ?", roleID, moduleID).First(&permission).Error
	return permission, err
}

func (r PermissionRepo) ListByRoleID(roleID int64) ([]data.Permission, error) {
	return r.List(roleID, 0)
}

func (r PermissionRepo) Create(permission *data.Permission) error {
	return db.MYSQLDB.Create(permission).Error
}

func (r PermissionRepo) Update(id int64, fields map[string]interface{}) error {
	result := db.MYSQLDB.Model(&data.Permission{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		exists, err := recordExists(db.MYSQLDB, &data.Permission{}, id)
		if err != nil {
			return err
		}
		if !exists {
			return gorm.ErrRecordNotFound
		}
	}
	return nil
}

func (r PermissionRepo) Delete(id int64) error {
	result := db.MYSQLDB.Delete(&data.Permission{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r PermissionRepo) DeleteByRoleAndModule(roleID, moduleID int64) error {
	result := db.MYSQLDB.Where("role_id = ? AND module_id = ?", roleID, moduleID).
		Delete(&data.Permission{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r PermissionRepo) ReplaceRolePermissions(roleID int64, permissions []data.Permission) error {
	return db.MYSQLDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&data.Permission{}).Error; err != nil {
			return err
		}
		if len(permissions) == 0 {
			return nil
		}
		return tx.Create(&permissions).Error
	})
}
