package device_auth

import (
	"sync"
)

// DeviceAuthsMap 设备认证映射表
type DeviceAuthsMap struct {
	devices map[string]bool
	mu      sync.RWMutex
}

// NewDeviceAuthsMap 创建新的设备认证映射表实例
func NewDeviceAuthsMap() *DeviceAuthsMap {
	return &DeviceAuthsMap{
		devices: make(map[string]bool),
	}
}

// Has 检查设备是否已认证
func (d *DeviceAuthsMap) Has(deviceUserName string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, exists := d.devices[deviceUserName]
	return exists
}

// Add 添加设备认证信息
func (d *DeviceAuthsMap) Add(deviceUserName string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.devices[deviceUserName] = true
}

// Remove 移除设备认证信息
func (d *DeviceAuthsMap) Remove(deviceUserName string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.devices, deviceUserName)
}

// Clear 清空所有设备认证信息
func (d *DeviceAuthsMap) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.devices = make(map[string]bool)
}

// Size 获取已认证设备数量
func (d *DeviceAuthsMap) Size() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.devices)
}
