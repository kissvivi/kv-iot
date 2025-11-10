package device_sync

import (
	"fmt"
	"kv-iot/datacenter/service/device_auth"
	"kv-iot/db"
	"log"
	"strings"
	"gorm.io/gorm"
)

// DeviceCenterClient 设备中心客户端接口
type DeviceCenterClient interface {
	GetAllDevices() ([]DeviceInfo, error)    // 获取所有设备
	GetDeviceByKeyAndNo(productKey, deviceNo string) (*DeviceInfo, error) // 获取单个设备
	GetProductsByDevice(productKey string) (*ProductInfo, error) // 获取产品信息
}

// DeviceCenterClientImpl 设备中心客户端实现
type DeviceCenterClientImpl struct {
	db *gorm.DB
}

// DeviceInfo 设备信息结构
type DeviceInfo struct {
	ProductKey  string
	DeviceNo    string
	ProductID   int64
	Name        string
	Description string
}

// ProductInfo 产品信息结构
type ProductInfo struct {
	ProductKey string
	Name       string
	ChannelID  string
	Description string
}

// NewDeviceCenterClient 创建设备中心客户端实例
func NewDeviceCenterClient(dbType string) *DeviceCenterClientImpl {
	// 直接使用全局的MySQLDB
	return &DeviceCenterClientImpl{
		db: db.MYSQLDB,
	}
}

// GetAllDevices 获取所有设备信息
func (c *DeviceCenterClientImpl) GetAllDevices() ([]DeviceInfo, error) {
	var devices []DeviceInfo
	
	// SQL查询，连接设备表和产品表，获取完整的设备信息
	query := `
		SELECT d.product_key, d.device_no, d.product_id, d.name, d.desc,
		       p.name as product_name, p.channel_id
		FROM devices d
		LEFT JOIN products p ON d.product_id = p.id
		WHERE d.deleted_at IS NULL
	`
	
	// 执行查询
	rows, err := c.db.Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询设备信息失败: %w", err)
	}
	defer rows.Close()
	
	// 处理结果
	for rows.Next() {
		var device DeviceInfo
		var productName, channelID string
		
		err := rows.Scan(
			&device.ProductKey, &device.DeviceNo, &device.ProductID,
			&device.Name, &device.Description, &productName, &channelID,
		)
		if err != nil {
			log.Printf("扫描设备数据失败: %v", err)
			continue
		}
		
		// 确保产品标识不为空
		if device.ProductKey == "" {
			log.Printf("设备 %s 的产品标识为空，跳过", device.DeviceNo)
			continue
		}
		
		devices = append(devices, device)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历设备结果失败: %w", err)
	}
	
	log.Printf("成功获取 %d 台设备信息", len(devices))
	return devices, nil
}

// GetDeviceByKeyAndNo 根据产品标识和设备编号获取单个设备信息
func (c *DeviceCenterClientImpl) GetDeviceByKeyAndNo(productKey, deviceNo string) (*DeviceInfo, error) {
	var device DeviceInfo
	
	// SQL查询
	query := `
		SELECT d.product_key, d.device_no, d.product_id, d.name, d.desc,
		       p.name as product_name, p.channel_id
		FROM devices d
		LEFT JOIN products p ON d.product_id = p.id
		WHERE d.product_key = ? AND d.device_no = ? AND d.deleted_at IS NULL
	`
	
	// 执行查询
	result := struct {
		ProductKey   string
		DeviceNo     string
		ProductID    int64
		DeviceName   string
		DeviceDesc   string
		ProductName  string
		ChannelID    string
	}{}
	err := c.db.Raw(query, productKey, deviceNo).Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}
	
	// 赋值到device结构体
	device.ProductKey = result.ProductKey
	device.DeviceNo = result.DeviceNo
	device.ProductID = result.ProductID
	device.Name = result.DeviceName
	device.Description = result.DeviceDesc
	
	return &device, nil
}

// GetProductsByDevice 根据产品标识获取产品信息
func (c *DeviceCenterClientImpl) GetProductsByDevice(productKey string) (*ProductInfo, error) {
	var product ProductInfo
	
	// SQL查询
	query := `
		SELECT product_key, name, channel_id, desc
		FROM products
		WHERE product_key = ? AND deleted_at IS NULL
	`
	
	// 执行查询
	result := struct {
		ProductKey   string
		ProductName  string
		ChannelID    string
		ProductDesc  string
	}{}
	err := c.db.Raw(query, productKey).Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("查询产品失败: %w", err)
	}
	
	// 赋值到product结构体
	product.ProductKey = result.ProductKey
	product.Name = result.ProductName
	product.ChannelID = result.ChannelID
	product.Description = result.ProductDesc
	
	return &product, nil
}

// SyncDevicesToAuthMap 同步设备信息到认证映射表
func SyncDevicesToAuthMap(authMap *device_auth.DeviceAuthsMap, client DeviceCenterClient) error {
	// 获取所有设备
	devices, err := client.GetAllDevices()
	if err != nil {
		return fmt.Errorf("获取设备列表失败: %w", err)
	}
	
	// 清空现有认证信息
	authMap.Clear()
	
	// 添加新的设备认证信息
	for _, device := range devices {
		// 跳过无效设备
		if strings.TrimSpace(device.ProductKey) == "" || strings.TrimSpace(device.DeviceNo) == "" {
			log.Printf("跳过无效设备: 产品标识='%s', 设备编号='%s'", device.ProductKey, device.DeviceNo)
			continue
		}
		
		deviceUserName := fmt.Sprintf("%s:%s", device.ProductKey, device.DeviceNo)
		authMap.Add(deviceUserName)
		log.Printf("已同步设备: %s (%s)", deviceUserName, device.Name)
	}
	
	log.Printf("设备同步完成，共同步 %d 台设备到认证系统", authMap.Size())
	return nil
}