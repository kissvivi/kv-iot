package repo

import (
	"kv-iot/db"
	"kv-iot/device/data"
	"time"
)

// DeviceConnRepo 设备连接仓库接口
type DeviceConnRepo interface {
	Add(conn data.DeviceConn) error
	Update(conn data.DeviceConn) error
	Delete(deviceID int64) error
	FindByDeviceNo(deviceNo string) (error, *data.DeviceConn)
	FindByProductKey(productKey string) (error, []data.DeviceConn)
	FindAll() (error, []data.DeviceConn)
	FindOnline() (error, []data.DeviceConn)
	UpdateHeartbeat(connID string) error
	UpdateStatus(deviceNo string, status int) error
}

// DeviceConnRepoImpl 设备连接仓库实现
type DeviceConnRepoImpl struct {
	db.BaseRepo[data.DeviceConn]
}

// NewDeviceConnRepo 创建设备连接仓库实例
func NewDeviceConnRepo() *DeviceConnRepoImpl {
	return &DeviceConnRepoImpl{}
}

// Add 添加设备连接记录
func (r *DeviceConnRepoImpl) Add(conn data.DeviceConn) error {
	return r.BaseRepo.Add(conn)
}

// Update 更新设备连接记录
func (r *DeviceConnRepoImpl) Update(conn data.DeviceConn) error {
	return r.BaseRepo.Update(conn)
}

// Delete 删除设备连接记录
func (r *DeviceConnRepoImpl) Delete(deviceID int64) error {
	conn := data.DeviceConn{}
	conn.DeviceID = deviceID
	return r.BaseRepo.Delete(conn)
}

// FindByDeviceNo 根据设备编号查询连接信息
func (r *DeviceConnRepoImpl) FindByDeviceNo(deviceNo string) (error, *data.DeviceConn) {
	conn := data.DeviceConn{}
	err := db.MYSQLDB.Where("device_no = ?", deviceNo).First(&conn).Error
	if err != nil {
		return err, nil
	}
	return nil, &conn
}

// FindByProductKey 根据产品标识查询连接信息
func (r *DeviceConnRepoImpl) FindByProductKey(productKey string) (error, []data.DeviceConn) {
	var conns []data.DeviceConn
	err := db.MYSQLDB.Where("product_key = ?", productKey).Find(&conns).Error
	return err, conns
}

// FindAll 查询所有连接信息
func (r *DeviceConnRepoImpl) FindAll() (error, []data.DeviceConn) {
	return r.BaseRepo.FindAll()
}

// FindOnline 查询在线设备连接
func (r *DeviceConnRepoImpl) FindOnline() (error, []data.DeviceConn) {
	var conns []data.DeviceConn
	err := db.MYSQLDB.Where("status = ?", 1).Find(&conns).Error
	return err, conns
}

// UpdateHeartbeat 更新心跳时间
func (r *DeviceConnRepoImpl) UpdateHeartbeat(connID string) error {
	return db.MYSQLDB.Model(&data.DeviceConn{}).Where("conn_id = ?", connID).Update("last_heartbeat", time.Now()).Error
}

// UpdateStatus 更新连接状态
func (r *DeviceConnRepoImpl) UpdateStatus(deviceNo string, status int) error {
	updateData := map[string]interface{}{
		"status": status,
	}
	if status == 0 { // 断开连接
		updateData["disconn_time"] = time.Now()
	}
	return db.MYSQLDB.Model(&data.DeviceConn{}).Where("device_no = ?", deviceNo).Updates(updateData).Error
}