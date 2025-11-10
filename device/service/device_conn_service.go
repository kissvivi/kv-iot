package service

import (
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
)

var _ deviceConnService = (*DeviceConnServiceImpl)(nil)

// deviceConnService 设备连接服务接口
type deviceConnService interface {
	// RegisterConn 注册设备连接
	RegisterConn(conn data.DeviceConn) error
	// UpdateConn 更新设备连接信息
	UpdateConn(conn data.DeviceConn) error
	// Disconnect 断开设备连接
	Disconnect(deviceNo string) error
	// GetConnByDeviceNo 根据设备编号获取连接信息
	GetConnByDeviceNo(deviceNo string) (error, *data.DeviceConn)
	// GetConnByProductKey 根据产品标识获取连接信息
	GetConnByProductKey(productKey string) (error, []data.DeviceConn)
	// GetAllConn 获取所有连接信息
	GetAllConn() (error, []data.DeviceConn)
	// GetOnlineConn 获取在线设备连接
	GetOnlineConn() (error, []data.DeviceConn)
	// UpdateHeartbeat 更新心跳时间
	UpdateHeartbeat(connID string) error
	// CheckAndCleanExpiredConn 检查并清理过期连接
	CheckAndCleanExpiredConn() error
}

// DeviceConnServiceImpl 设备连接服务实现
type DeviceConnServiceImpl struct {
	deviceConnRepo repo.DeviceConnRepo
}

// NewDeviceConnServiceImpl 创建设备连接服务实例
func NewDeviceConnServiceImpl(deviceConnRepo repo.DeviceConnRepo) *DeviceConnServiceImpl {
	return &DeviceConnServiceImpl{deviceConnRepo: deviceConnRepo}
}

// RegisterConn 注册设备连接
func (s *DeviceConnServiceImpl) RegisterConn(conn data.DeviceConn) error {
	return s.deviceConnRepo.Add(conn)
}

// UpdateConn 更新设备连接信息
func (s *DeviceConnServiceImpl) UpdateConn(conn data.DeviceConn) error {
	return s.deviceConnRepo.Update(conn)
}

// Disconnect 断开设备连接
func (s *DeviceConnServiceImpl) Disconnect(deviceNo string) error {
	return s.deviceConnRepo.UpdateStatus(deviceNo, 0) // 0表示断开
}

// GetConnByDeviceNo 根据设备编号获取连接信息
func (s *DeviceConnServiceImpl) GetConnByDeviceNo(deviceNo string) (error, *data.DeviceConn) {
	return s.deviceConnRepo.FindByDeviceNo(deviceNo)
}

// GetConnByProductKey 根据产品标识获取连接信息
func (s *DeviceConnServiceImpl) GetConnByProductKey(productKey string) (error, []data.DeviceConn) {
	return s.deviceConnRepo.FindByProductKey(productKey)
}

// GetAllConn 获取所有连接信息
func (s *DeviceConnServiceImpl) GetAllConn() (error, []data.DeviceConn) {
	return s.deviceConnRepo.FindAll()
}

// GetOnlineConn 获取在线设备连接
func (s *DeviceConnServiceImpl) GetOnlineConn() (error, []data.DeviceConn) {
	return s.deviceConnRepo.FindOnline()
}

// UpdateHeartbeat 更新心跳时间
func (s *DeviceConnServiceImpl) UpdateHeartbeat(connID string) error {
	return s.deviceConnRepo.UpdateHeartbeat(connID)
}

// CheckAndCleanExpiredConn 检查并清理过期连接
func (s *DeviceConnServiceImpl) CheckAndCleanExpiredConn() error {
	// 这里可以实现心跳超时检查逻辑
	// 目前返回nil，后续可以扩展
	return nil
}