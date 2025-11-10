package data

import (
	"gorm.io/gorm"
	"time"
)

// ChannelType 通道类型枚举
const (
	ChannelTypeMQTT      = "mqtt"
	ChannelTypeTCP       = "tcp"
	ChannelTypeModbusTCP = "modbus_tcp"
	ChannelTypeHTTP      = "http"
	ChannelTypeWebSocket = "websocket"
)

// ChannelStatus 通道状态枚举
const (
	ChannelStatusDisabled = iota // 禁用
	ChannelStatusEnabled         // 启用
	ChannelStatusRunning         // 运行中
	ChannelStatusError           // 错误
)

// ChannelInstance 通道实例 - 代表实际运行的通信通道
type ChannelInstance struct {
	gorm.Model
	ChannelID     string    `json:"channel_id" gorm:"column:channel_id"`       // 关联的通道模板ID
	ProductID     int64     `json:"product_id" gorm:"column:product_id"`       // 关联的产品ID
	ProductKey    string    `json:"product_key" gorm:"column:product_key"`     // 产品标识
	InstanceName  string    `json:"instance_name" gorm:"column:instance_name"` // 实例名称
	ChannelType   string    `json:"channel_type" gorm:"column:channel_type"`   // 通道类型(mqtt/tcp/modbus_tcp等)
	Config        string    `json:"config" gorm:"column:config"`               // 通道配置(JSON格式)
	Status        int       `json:"status" gorm:"column:status"`               // 通道状态
	RunningPort   string    `json:"running_port" gorm:"column:running_port"`   // 运行端口
	StartUpTime   time.Time `json:"start_up_time" gorm:"column:start_up_time"` // 启动时间
	LastHeartbeat time.Time `json:"last_heartbeat" gorm:"column:last_heartbeat"` // 最后心跳时间
	ErrorMsg      string    `json:"error_msg" gorm:"column:error_msg"`         // 错误信息
	MaxConn       int       `json:"max_conn" gorm:"column:max_conn"`           // 最大连接数
	CurrentConn   int       `json:"current_conn" gorm:"column:current_conn"`   // 当前连接数
}

// TableName 设置表名
func (m *ChannelInstance) TableName() string {
	return "channel_instances"
}

// ChannelInstanceRepo 通道实例仓库接口
type ChannelInstanceRepo interface {
	Add(instance ChannelInstance) error
	Update(instance ChannelInstance) error
	Delete(id int64) error
	FindByID(id int64) (error, *ChannelInstance)
	FindByProductID(productID int64) (error, []ChannelInstance)
	FindByProductKey(productKey string) (error, *ChannelInstance)
	FindByStatus(status int) (error, []ChannelInstance)
	FindAll() (error, []ChannelInstance)
	UpdateStatus(id int64, status int) error
	UpdateConnectionCount(id int64, increment bool) error
}