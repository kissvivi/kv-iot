package repo

import (
	"kv-iot/device/data"

	"gorm.io/gorm"
)

type DevicePointBindingRepoImpl struct {
	db *gorm.DB
}

var _ data.DevicePointBindingRepo = (*DevicePointBindingRepoImpl)(nil)

func NewDevicePointBindingRepo(db *gorm.DB) data.DevicePointBindingRepo {
	return &DevicePointBindingRepoImpl{db: db}
}

func (r *DevicePointBindingRepoImpl) Add(binding data.DevicePointBinding) error {
	return r.db.Create(&binding).Error
}

func (r *DevicePointBindingRepoImpl) Update(binding data.DevicePointBinding) error {
	return r.db.Save(&binding).Error
}

func (r *DevicePointBindingRepoImpl) Delete(id int64) error {
	return r.db.Delete(&data.DevicePointBinding{}, id).Error
}

func (r *DevicePointBindingRepoImpl) FindByID(id int64) (error, *data.DevicePointBinding) {
	var binding data.DevicePointBinding
	if err := r.db.First(&binding, id).Error; err != nil {
		return err, nil
	}
	return nil, &binding
}

func (r *DevicePointBindingRepoImpl) FindByDeviceID(deviceID int64) (error, []data.DevicePointBinding) {
	var bindings []data.DevicePointBinding
	if err := r.db.Where("device_id = ?", deviceID).Order("id").Find(&bindings).Error; err != nil {
		return err, nil
	}
	return nil, bindings
}

func (r *DevicePointBindingRepoImpl) FindByChannelInstanceID(channelInstanceID int64) (error, []data.DevicePointBinding) {
	var bindings []data.DevicePointBinding
	err := r.db.Where("channel_instance_id = ?", channelInstanceID).Order("device_id, id").Find(&bindings).Error
	if err != nil {
		return err, nil
	}
	return nil, bindings
}

func (r *DevicePointBindingRepoImpl) FindByProductPointBindingID(productPointBindingID int64) (error, []data.DevicePointBinding) {
	var bindings []data.DevicePointBinding
	err := r.db.Where("product_point_binding_id = ?", productPointBindingID).
		Order("device_id, id").
		Find(&bindings).Error
	if err != nil {
		return err, nil
	}
	return nil, bindings
}

func (r *DevicePointBindingRepoImpl) FindByInstance(deviceID int64, channelInstanceID int64, productPointBindingID int64) (error, *data.DevicePointBinding) {
	var binding data.DevicePointBinding
	err := r.db.Where(
		"device_id = ? AND channel_instance_id = ? AND product_point_binding_id = ?",
		deviceID,
		channelInstanceID,
		productPointBindingID,
	).First(&binding).Error
	if err != nil {
		return err, nil
	}
	return nil, &binding
}

func (r *DevicePointBindingRepoImpl) FindAll() (error, []data.DevicePointBinding) {
	var bindings []data.DevicePointBinding
	if err := r.db.Order("device_id, channel_instance_id, product_point_binding_id, id").Find(&bindings).Error; err != nil {
		return err, nil
	}
	return nil, bindings
}

func (r *DevicePointBindingRepoImpl) UpdateEnabled(id int64, enabled bool) error {
	return r.db.Model(&data.DevicePointBinding{}).Where("id = ?", id).Update("enabled", enabled).Error
}
