package data

import "gorm.io/gorm"

// DevicePointBinding binds a product point protocol definition to a device instance.
type DevicePointBinding struct {
	gorm.Model
	DeviceID              int64  `json:"device_id" gorm:"column:device_id;not null;index:idx_device_point_bindings_device_id;uniqueIndex:uk_device_point_binding_instance,priority:1"`
	ChannelInstanceID     int64  `json:"channel_instance_id" gorm:"column:channel_instance_id;not null;index:idx_device_point_bindings_channel_instance_id;uniqueIndex:uk_device_point_binding_instance,priority:3"`
	ProductPointBindingID int64  `json:"product_point_binding_id" gorm:"column:product_point_binding_id;not null;index:idx_device_point_bindings_product_point_binding_id;uniqueIndex:uk_device_point_binding_instance,priority:2"`
	BindingOverride       string `json:"binding_override" gorm:"column:binding_override;type:text"`
	Enabled               bool   `json:"enabled" gorm:"column:enabled;not null;index:idx_device_point_bindings_enabled"`
}

func (m *DevicePointBinding) TableName() string {
	return "device_point_bindings"
}

type DevicePointBindingRepo interface {
	Add(binding DevicePointBinding) error
	Update(binding DevicePointBinding) error
	Delete(id int64) error
	FindByID(id int64) (error, *DevicePointBinding)
	FindByDeviceID(deviceID int64) (error, []DevicePointBinding)
	FindByChannelInstanceID(channelInstanceID int64) (error, []DevicePointBinding)
	FindByProductPointBindingID(productPointBindingID int64) (error, []DevicePointBinding)
	FindByInstance(deviceID int64, channelInstanceID int64, productPointBindingID int64) (error, *DevicePointBinding)
	FindAll() (error, []DevicePointBinding)
	UpdateEnabled(id int64, enabled bool) error
}
