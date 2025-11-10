package repo

import (
	"kv-iot/db"
	"kv-iot/device/data"
)

// DevicesRepo 设备仓库实现
type DevicesRepo struct {
	db.BaseRepo[data.Devices]
}

// NewDevicesRepo 创建新的设备仓库实例
func NewDevicesRepo() *DevicesRepo {
	// BaseRepo已经是DevicesRepo的匿名字段，不需要再初始化
	return &DevicesRepo{}
}

// Add 添加设备记录
func (r *DevicesRepo) Add(devices data.Devices) error {
	return r.BaseRepo.Add(devices)
}

// Delete 删除设备记录
func (r *DevicesRepo) Delete(devices data.Devices) error {
	return r.BaseRepo.Delete(devices)
}

// FindAll 查询所有设备
func (r *DevicesRepo) FindAll() (error, []data.Devices) {
	err, devices := r.BaseRepo.FindAll()
	return err, devices
}

// FindByStruct 根据条件查询设备
func (r *DevicesRepo) FindByStruct(devices data.Devices) (error, []data.Devices) {
	err, result := r.BaseRepo.FindByStruct(devices)
	return err, result
}
