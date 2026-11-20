package repo

import (
	"kv-iot/device/data"

	"gorm.io/gorm"
)

type ProductProtocolProfileRepoImpl struct {
	db *gorm.DB
}

var _ data.ProductProtocolProfileRepo = (*ProductProtocolProfileRepoImpl)(nil)

func NewProductProtocolProfileRepo(db *gorm.DB) data.ProductProtocolProfileRepo {
	return &ProductProtocolProfileRepoImpl{db: db}
}

func (r *ProductProtocolProfileRepoImpl) Add(profile data.ProductProtocolProfile) error {
	return r.db.Create(&profile).Error
}

func (r *ProductProtocolProfileRepoImpl) Update(profile data.ProductProtocolProfile) error {
	return r.db.Save(&profile).Error
}

func (r *ProductProtocolProfileRepoImpl) Delete(id int64) error {
	return r.db.Delete(&data.ProductProtocolProfile{}, id).Error
}

func (r *ProductProtocolProfileRepoImpl) FindByID(id int64) (error, *data.ProductProtocolProfile) {
	var profile data.ProductProtocolProfile
	if err := r.db.First(&profile, id).Error; err != nil {
		return err, nil
	}
	return nil, &profile
}

func (r *ProductProtocolProfileRepoImpl) FindByProductID(productID int64) (error, []data.ProductProtocolProfile) {
	var profiles []data.ProductProtocolProfile
	if err := r.db.Where("product_id = ?", productID).Order("id").Find(&profiles).Error; err != nil {
		return err, nil
	}
	return nil, profiles
}

func (r *ProductProtocolProfileRepoImpl) FindByProductIDAndVersion(productID int64, version string) (error, *data.ProductProtocolProfile) {
	var profile data.ProductProtocolProfile
	err := r.db.Where("product_id = ? AND profile_version = ?", productID, version).First(&profile).Error
	if err != nil {
		return err, nil
	}
	return nil, &profile
}

func (r *ProductProtocolProfileRepoImpl) FindEnabledByProductID(productID int64) (error, []data.ProductProtocolProfile) {
	var profiles []data.ProductProtocolProfile
	err := r.db.Where("product_id = ? AND status = ?", productID, data.ProtocolProfileStatusEnabled).
		Order("id").
		Find(&profiles).Error
	if err != nil {
		return err, nil
	}
	return nil, profiles
}

func (r *ProductProtocolProfileRepoImpl) FindAll() (error, []data.ProductProtocolProfile) {
	var profiles []data.ProductProtocolProfile
	if err := r.db.Order("id").Find(&profiles).Error; err != nil {
		return err, nil
	}
	return nil, profiles
}

func (r *ProductProtocolProfileRepoImpl) UpdateStatus(id int64, status string) error {
	return r.db.Model(&data.ProductProtocolProfile{}).Where("id = ?", id).Update("status", status).Error
}
