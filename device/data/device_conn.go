package data

import (
	"gorm.io/gorm"
	"time"
)

// DeviceConn 设备连接信息
type DeviceConn struct {
	gorm.Model
	DeviceID      int64     `json:"device_id" gorm:"column:device_id"`           // 设备ID
	DeviceNo      string    `json:"device_no" gorm:"column:device_no"`           // 设备编号
	ProductID     int64     `json:"product_id" gorm:"column:product_id"`         // 产品ID
	ProductKey    string    `json:"product_key" gorm:"column:product_key"`       // 产品标识
	ConnID        string    `json:"conn_id" gorm:"column:conn_id"`               // 连接ID
	ClientID      string    `json:"client_id" gorm:"column:client_id"`           // 客户端ID
	ConnType      string    `json:"conn_type" gorm:"column:conn_type"`           // 连接类型(mqtt/http/tcp)
	IP            string    `json:"ip" gorm:"column:ip"`                         // 设备IP
	Port          string    `json:"port" gorm:"column:port"`                     // 设备端口
	Status        int       `json:"status" gorm:"column:status"`                 // 连接状态(0:断开,1:连接)
	LastHeartbeat time.Time `json:"last_heartbeat" gorm:"column:last_heartbeat"` // 最后心跳时间
	ConnTime      time.Time `json:"conn_time" gorm:"column:conn_time"`           // 连接时间
	DisconnTime   time.Time `json:"disconn_time" gorm:"column:disconn_time"`     // 断开时间
	ChannelID     string    `json:"channel_id" gorm:"column:channel_id"`         // 通讯通道ID
}

// TableName 设置表名
func (m *DeviceConn) TableName() string {
	return "device_conn"
}

// DeviceConnRepo 设备连接仓库接口
type DeviceConnRepo interface {
	Add(conn DeviceConn) error
	Update(conn DeviceConn) error
	Delete(deviceID int64) error
	FindByDeviceNo(deviceNo string) (error, *DeviceConn)
	FindByProductKey(productKey string) (error, []DeviceConn)
	FindAll() (error, []DeviceConn)
	FindOnline() (error, []DeviceConn)
	UpdateHeartbeat(connID string) error
	UpdateStatus(deviceNo string, status int) error
}