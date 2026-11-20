package repo

import (
	"kv-iot/auth/data"
	"kv-iot/db"

	"gorm.io/gorm"
)

type UserRepo struct {
	data.AuthRepo[data.User]
}

func (r UserRepo) List() ([]data.User, error) {
	users := make([]data.User, 0)
	err := db.MYSQLDB.Order("id ASC").Find(&users).Error
	return users, err
}

func (r UserRepo) Get(id int64) (data.User, error) {
	var user data.User
	err := db.MYSQLDB.First(&user, id).Error
	return user, err
}

func (r UserRepo) FindByUserName(userName string) (data.User, error) {
	var user data.User
	err := db.MYSQLDB.Where("user_name = ?", userName).First(&user).Error
	return user, err
}

func (r UserRepo) CreateWithRoles(user *data.User, roleIDs []int64) error {
	return db.MYSQLDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		return replaceUserRoles(tx, user.ID, roleIDs)
	})
}

func (r UserRepo) UpdateWithRoles(id int64, fields map[string]interface{}, roleIDs *[]int64) error {
	return db.MYSQLDB.Transaction(func(tx *gorm.DB) error {
		if len(fields) > 0 {
			result := tx.Model(&data.User{}).Where("id = ?", id).Updates(fields)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				exists, err := recordExists(tx, &data.User{}, id)
				if err != nil {
					return err
				}
				if !exists {
					return gorm.ErrRecordNotFound
				}
			}
		}
		if roleIDs != nil {
			return replaceUserRoles(tx, uint(id), *roleIDs)
		}
		return nil
	})
}

func (r UserRepo) UpdatePassword(id int64, passwordHash string) error {
	result := db.MYSQLDB.Model(&data.User{}).
		Where("id = ?", id).
		Update("password", passwordHash)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r UserRepo) DeleteWithRelations(id int64) error {
	return db.MYSQLDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&data.UserRole{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&data.User{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func replaceUserRoles(tx *gorm.DB, userID uint, roleIDs []int64) error {
	if err := tx.Where("user_id = ?", userID).Delete(&data.UserRole{}).Error; err != nil {
		return err
	}
	roleIDs = uniqueInt64(roleIDs)
	if len(roleIDs) == 0 {
		return nil
	}
	rows := make([]data.UserRole, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		rows = append(rows, data.UserRole{
			UserID: int64(userID),
			RoleID: roleID,
		})
	}
	return tx.Table("user_role").Create(&rows).Error
}

func uniqueInt64(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
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

func recordExists(tx *gorm.DB, model interface{}, id int64) (bool, error) {
	var count int64
	if err := tx.Model(model).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
