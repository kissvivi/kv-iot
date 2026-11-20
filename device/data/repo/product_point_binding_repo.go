package repo

import (
	"kv-iot/device/data"

	"gorm.io/gorm"
)

type ProductPointBindingRepoImpl struct {
	db *gorm.DB
}

var _ data.ProductPointBindingRepo = (*ProductPointBindingRepoImpl)(nil)

func NewProductPointBindingRepo(db *gorm.DB) data.ProductPointBindingRepo {
	return &ProductPointBindingRepoImpl{db: db}
}

func (r *ProductPointBindingRepoImpl) Add(binding data.ProductPointBinding) error {
	return r.db.Create(&binding).Error
}

func (r *ProductPointBindingRepoImpl) Update(binding data.ProductPointBinding) error {
	return r.db.Save(&binding).Error
}

func (r *ProductPointBindingRepoImpl) Delete(id int64) error {
	return r.db.Delete(&data.ProductPointBinding{}, id).Error
}

func (r *ProductPointBindingRepoImpl) FindByID(id int64) (error, *data.ProductPointBinding) {
	var binding data.ProductPointBinding
	if err := r.db.First(&binding, id).Error; err != nil {
		return err, nil
	}
	return nil, &binding
}

func (r *ProductPointBindingRepoImpl) FindByProfileID(profileID int64) (error, []data.ProductPointBinding) {
	var bindings []data.ProductPointBinding
	if err := r.db.Where("profile_id = ?", profileID).Order("point_kind, point_id, id").Find(&bindings).Error; err != nil {
		return err, nil
	}
	return nil, bindings
}

func (r *ProductPointBindingRepoImpl) FindByProfileIDAndPointKind(profileID int64, pointKind string) (error, []data.ProductPointBinding) {
	var bindings []data.ProductPointBinding
	err := r.db.Where("profile_id = ? AND point_kind = ?", profileID, pointKind).
		Order("point_id, id").
		Find(&bindings).Error
	if err != nil {
		return err, nil
	}
	return nil, bindings
}

func (r *ProductPointBindingRepoImpl) FindByProfileIDAndPoint(profileID int64, pointKind string, pointID int64) (error, *data.ProductPointBinding) {
	var binding data.ProductPointBinding
	err := r.db.Where(
		"profile_id = ? AND point_kind = ? AND point_id = ?",
		profileID,
		pointKind,
		pointID,
	).First(&binding).Error
	if err != nil {
		return err, nil
	}
	return nil, &binding
}

func (r *ProductPointBindingRepoImpl) FindAll() (error, []data.ProductPointBinding) {
	var bindings []data.ProductPointBinding
	if err := r.db.Order("profile_id, point_kind, point_id, id").Find(&bindings).Error; err != nil {
		return err, nil
	}
	return nil, bindings
}

func (r *ProductPointBindingRepoImpl) UpdateEnabled(id int64, enabled bool) error {
	return r.db.Model(&data.ProductPointBinding{}).Where("id = ?", id).Update("enabled", enabled).Error
}
