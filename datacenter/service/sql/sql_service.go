package sql

import (
	"fmt"
	"kv-iot/datacenter/data"
	"kv-iot/db"
	"log"
	"time"
)

// SqlService 提供SQL数据库操作功能
type SqlService struct {
	// 不再直接存储db实例，而是使用全局变量
}

// NewSqlService 创建新的SQL服务实例
func NewSqlService() *SqlService {
	// 检查全局MySQL数据库连接是否存在
	if db.MYSQLDB == nil {
		log.Printf("警告: MySQL数据库连接未初始化，请确保在调用此服务前已初始化数据库")
	}

	return &SqlService{}
}

// Store 存储设备消息到SQL数据库
func (s *SqlService) Store(msg data.KvMsg) error {
	if msg.Device.DeviceNo == "" {
		return fmt.Errorf("设备编号为空，无法存储数据")
	}

	// 存储设备信息
	if err := s.saveDeviceInfo(msg); err != nil {
		log.Printf("存储设备信息失败: %v", err)
		// 不中断流程，继续尝试存储数据
	}

	// 存储设备数据
	if err := s.saveDeviceData(msg); err != nil {
		return fmt.Errorf("存储设备数据失败: %w", err)
	}

	return nil
}

// saveDeviceInfo 保存设备信息
func (s *SqlService) saveDeviceInfo(msg data.KvMsg) error {
	// 准备SQL语句 - 这里使用UPSERT操作
	query := `
		INSERT INTO devices (product_key, device_no, device_name, last_online_time)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			last_online_time = VALUES(last_online_time),
			device_name = COALESCE(VALUES(device_name), device_name)
	`

	// 获取产品键的字符串表示
	productKeyStr := fmt.Sprintf("%v", msg.Device.ProductKey)

	// 使用GORM全局实例执行SQL
	if db.MYSQLDB != nil {
		err := db.MYSQLDB.Exec(query,
			productKeyStr,
			msg.Device.DeviceNo,
			msg.Device.Name, // 直接使用字符串，不需要nil检查
			time.Now(),
		).Error

		if err != nil {
			return fmt.Errorf("保存设备信息失败: %w", err)
		}

		log.Printf("设备信息已更新: %s", msg.Device.DeviceNo)
	} else {
		return fmt.Errorf("数据库连接未初始化")
	}
	return nil
}

// saveDeviceData 保存设备数据
func (s *SqlService) saveDeviceData(msg data.KvMsg) error {
	// 处理属性数据
	if err := s.savePropertyData(msg); err != nil {
		return err
	}

	// 处理事件数据
	if err := s.saveEventData(msg); err != nil {
		return err
	}

	return nil
}

// savePropertyData 保存属性数据
func (s *SqlService) savePropertyData(msg data.KvMsg) error {
	// 只有当属性标识符不为空时才保存
	if msg.Property.Identifier != "" {
		query := `
			INSERT INTO device_properties (device_no, property_id, property_name, property_value, property_type, update_time)
			VALUES (?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				property_value = VALUES(property_value),
				update_time = VALUES(update_time)
		`

		// 使用GORM全局实例执行SQL
		if db.MYSQLDB != nil {
			err := db.MYSQLDB.Exec(query,
				msg.Device.DeviceNo,
				msg.Property.Identifier,
				msg.Property.Name,
				fmt.Sprintf("%v", msg.Property.Value),
				msg.Property.DataType,
				time.Now(),
			).Error

			if err != nil {
				return fmt.Errorf("保存属性数据失败: %w", err)
			}

			log.Printf("属性数据已保存: 设备=%s, 属性=%s", msg.Device.DeviceNo, msg.Property.Identifier)
		} else {
			return fmt.Errorf("数据库连接未初始化")
		}
	}

	return nil
}

// saveEventData 保存事件数据
func (s *SqlService) saveEventData(msg data.KvMsg) error {
	// 只有当事件标识符不为空时才保存
	if msg.Event.Identifier != "" {
		// 准备SQL语句
		query := `
			INSERT INTO device_events (device_no, event_id, event_name, event_time, event_data)
			VALUES (?, ?, ?, ?, ?)
		`

		// 将整个事件对象序列化为JSON字符串
		eventData := fmt.Sprintf("%+v", msg.Event)

		// 使用GORM全局实例执行SQL
		if db.MYSQLDB != nil {
			err := db.MYSQLDB.Exec(query,
				msg.Device.DeviceNo,
				msg.Event.Identifier,
				msg.Event.Name,
				time.Now(),
				eventData,
			).Error

			if err != nil {
				return fmt.Errorf("保存事件数据失败: %w", err)
			}

			log.Printf("事件数据已保存: 设备=%s, 事件=%s", msg.Device.DeviceNo, msg.Event.Identifier)
		} else {
			return fmt.Errorf("数据库连接未初始化")
		}
	}

	return nil
}

// Query 查询设备数据
func (s *SqlService) Query(query string, args ...interface{}) (interface{}, error) {
	// 使用GORM全局实例执行查询
	if db.MYSQLDB != nil {
		rows, err := db.MYSQLDB.Raw(query, args...).Rows()
		if err != nil {
			return nil, fmt.Errorf("查询失败: %w", err)
		}
		defer rows.Close()

		// 这里简化处理，实际应用中需要根据具体查询结果结构解析
		log.Printf("执行查询: %s", query)
		return rows, nil
	}
	return nil, fmt.Errorf("数据库连接未初始化")
}
