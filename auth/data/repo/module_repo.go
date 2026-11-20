package repo

import (
	"kv-iot/auth/data"
	"kv-iot/db"

	"gorm.io/gorm"
)

type ModuleRepo struct {
	data.AuthRepo[data.Module]
}

func (r ModuleRepo) List() ([]data.Module, error) {
	modules := make([]data.Module, 0)
	err := db.MYSQLDB.Order("parent_id ASC, id ASC").Find(&modules).Error
	return modules, err
}

func (r ModuleRepo) Get(id int64) (data.Module, error) {
	var module data.Module
	err := db.MYSQLDB.First(&module, id).Error
	return module, err
}

func (r ModuleRepo) FindByCode(code string) (data.Module, error) {
	var module data.Module
	err := db.MYSQLDB.Where("code = ?", code).First(&module).Error
	return module, err
}

func (r ModuleRepo) FindByIDs(ids []int64) ([]data.Module, error) {
	modules := make([]data.Module, 0)
	if len(ids) == 0 {
		return modules, nil
	}
	err := db.MYSQLDB.Where("id IN ?", ids).Order("id ASC").Find(&modules).Error
	return modules, err
}

func (r ModuleRepo) Create(module *data.Module) error {
	return db.MYSQLDB.Create(module).Error
}

func (r ModuleRepo) Update(id int64, fields map[string]interface{}) error {
	result := db.MYSQLDB.Model(&data.Module{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		exists, err := recordExists(db.MYSQLDB, &data.Module{}, id)
		if err != nil {
			return err
		}
		if !exists {
			return gorm.ErrRecordNotFound
		}
	}
	return nil
}

func (r ModuleRepo) DeleteWithRelations(id int64) error {
	return db.MYSQLDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("module_id = ?", id).Delete(&data.Permission{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&data.Module{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
