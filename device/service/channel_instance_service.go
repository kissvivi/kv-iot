package service

import (
	"encoding/json"
	"errors"
	"kv-iot/device/data"
	"log"
	"strings"
	"sync"
	"time"
)

// ChannelInstanceService 通道实例服务接口
type ChannelInstanceService interface {
	// 创建通道实例
	CreateChannelInstance(instance *data.ChannelInstance) error
	// 根据ID获取通道实例
	GetChannelInstanceByID(id int64) (error, *data.ChannelInstance)
	// 根据产品ID获取通道实例
	GetChannelInstancesByProductID(productID int64) (error, []data.ChannelInstance)
	// 获取所有通道实例
	GetAllChannelInstances() (error, []data.ChannelInstance)
	// 更新通道实例
	UpdateChannelInstance(id int64, update ChannelInstanceUpdate) (*data.ChannelInstance, error)
	// 删除通道实例
	DeleteChannelInstance(id int64) error
	// 启动通道实例
	StartChannelInstance(id int64) error
	// 停止通道实例
	StopChannelInstance(id int64) error
	// 重启通道实例
	RestartChannelInstance(id int64) error
	// 获取通道实例状态
	GetChannelInstanceStatus(id int64) (int, error)
	// 更新通道实例心跳
	UpdateChannelInstanceHeartbeat(id int64) error
	// 增加连接计数
	IncreaseConnectionCount(id int64) error
	// 减少连接计数
	DecreaseConnectionCount(id int64) error
}

type ChannelInstanceUpdate struct {
	ChannelID    *string `json:"channel_id"`
	ProductID    *int64  `json:"product_id"`
	ProductKey   *string `json:"product_key"`
	InstanceName *string `json:"instance_name"`
	ChannelType  *string `json:"channel_type"`
	Config       *string `json:"config"`
	Status       *int    `json:"status"`
	MaxConn      *int    `json:"max_conn"`
	Enabled      *bool   `json:"enabled"`
}

// ChannelInstanceServiceImpl 通道实例服务实现
type ChannelInstanceServiceImpl struct {
	channelInstanceRepo data.ChannelInstanceRepo
	// 用于存储运行中的通道实例
	runningInstances sync.Map
	// 互斥锁，用于保护并发访问
	mutex sync.RWMutex
}

// NewChannelInstanceServiceImpl 创建通道实例服务
func NewChannelInstanceServiceImpl(channelInstanceRepo data.ChannelInstanceRepo) ChannelInstanceService {
	return &ChannelInstanceServiceImpl{
		channelInstanceRepo: channelInstanceRepo,
		runningInstances:    sync.Map{},
	}
}

// CreateChannelInstance 创建通道实例
func (s *ChannelInstanceServiceImpl) CreateChannelInstance(instance *data.ChannelInstance) error {
	// 验证通道配置
	if err := s.validateChannelConfig(instance.ChannelType, instance.Config); err != nil {
		return err
	}

	// 设置默认值
	instance.Status = data.ChannelStatusDisabled
	instance.CurrentConn = 0
	instance.Enabled = true
	now := time.Now()
	instance.StartUpTime = now
	instance.LastHeartbeat = now

	// 保存到数据库
	return s.channelInstanceRepo.Add(*instance)
}

// GetChannelInstanceByID 根据ID获取通道实例
func (s *ChannelInstanceServiceImpl) GetChannelInstanceByID(id int64) (error, *data.ChannelInstance) {
	return s.channelInstanceRepo.FindByID(id)
}

// GetChannelInstancesByProductID 根据产品ID获取通道实例
func (s *ChannelInstanceServiceImpl) GetChannelInstancesByProductID(productID int64) (error, []data.ChannelInstance) {
	return s.channelInstanceRepo.FindByProductID(productID)
}

// GetAllChannelInstances 获取所有通道实例
func (s *ChannelInstanceServiceImpl) GetAllChannelInstances() (error, []data.ChannelInstance) {
	return s.channelInstanceRepo.FindAll()
}

// UpdateChannelInstance 更新通道实例
func (s *ChannelInstanceServiceImpl) UpdateChannelInstance(id int64, update ChannelInstanceUpdate) (*data.ChannelInstance, error) {
	err, instance := s.channelInstanceRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.New("channel instance not found")
	}
	if update.ChannelID != nil {
		instance.ChannelID = strings.TrimSpace(*update.ChannelID)
	}
	if update.ProductID != nil {
		instance.ProductID = *update.ProductID
	}
	if update.ProductKey != nil {
		instance.ProductKey = strings.TrimSpace(*update.ProductKey)
	}
	if update.InstanceName != nil {
		instance.InstanceName = strings.TrimSpace(*update.InstanceName)
	}
	if update.ChannelType != nil {
		instance.ChannelType = strings.TrimSpace(*update.ChannelType)
	}
	if update.Config != nil {
		instance.Config = *update.Config
	}
	if update.Status != nil {
		instance.Status = *update.Status
	}
	if update.MaxConn != nil {
		instance.MaxConn = *update.MaxConn
	}
	if update.Enabled != nil {
		instance.Enabled = *update.Enabled
	}
	if instance.InstanceName == "" {
		return nil, errors.New("channel instance name is required")
	}
	if err := s.validateChannelConfig(instance.ChannelType, instance.Config); err != nil {
		return nil, err
	}
	if err := s.channelInstanceRepo.Update(*instance); err != nil {
		return nil, err
	}
	return instance, nil
}

// DeleteChannelInstance 删除通道实例
func (s *ChannelInstanceServiceImpl) DeleteChannelInstance(id int64) error {
	// 获取通道实例
	err, instance := s.channelInstanceRepo.FindByID(id)
	if err != nil {
		return err
	}

	// 如果通道正在运行，需要先停止
	if instance.Status == data.ChannelStatusRunning {
		if err := s.StopChannelInstance(id); err != nil {
			return err
		}
	}

	// 删除从内存中
	s.runningInstances.Delete(id)

	// 从数据库中删除
	return s.channelInstanceRepo.Delete(id)
}

// StartChannelInstance 启动通道实例
func (s *ChannelInstanceServiceImpl) StartChannelInstance(id int64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 获取通道实例
	err, instance := s.channelInstanceRepo.FindByID(id)
	if err != nil {
		return err
	}

	// 检查状态
	if instance.Status == data.ChannelStatusRunning {
		return errors.New("channel instance already running")
	}

	// 检查是否已经在内存中
	if _, exists := s.runningInstances.Load(id); exists {
		return errors.New("channel instance already in memory")
	}

	// 启动通道服务（这里是模拟，实际应该根据不同的通道类型启动对应的服务）
	// 在实际实现中，这里会根据ChannelType创建对应的MQTT服务、TCP服务等
	log.Printf("Starting channel instance %d of type %s", id, instance.ChannelType)

	// 更新状态
	instance.Status = data.ChannelStatusRunning
	instance.Enabled = true
	instance.StartUpTime = time.Now()
	instance.LastHeartbeat = time.Now()
	instance.ErrorMsg = ""

	// 保存到数据库
	if err := s.channelInstanceRepo.Update(*instance); err != nil {
		return err
	}

	// 保存到内存
	s.runningInstances.Store(id, instance)

	return nil
}

// StopChannelInstance 停止通道实例
func (s *ChannelInstanceServiceImpl) StopChannelInstance(id int64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 获取通道实例
	err, instance := s.channelInstanceRepo.FindByID(id)
	if err != nil {
		return err
	}

	// 检查状态
	if instance.Status == data.ChannelStatusDisabled {
		return nil
	}

	// 停止通道服务（这里是模拟，实际应该根据不同的通道类型停止对应的服务）
	log.Printf("Stopping channel instance %d", id)

	// 更新状态
	instance.Status = data.ChannelStatusDisabled
	instance.Enabled = false
	instance.CurrentConn = 0

	// 保存到数据库
	if err := s.channelInstanceRepo.Update(*instance); err != nil {
		return err
	}

	// 从内存中移除
	s.runningInstances.Delete(id)

	return nil
}

// RestartChannelInstance 重启通道实例
func (s *ChannelInstanceServiceImpl) RestartChannelInstance(id int64) error {
	// 先停止
	if err := s.StopChannelInstance(id); err != nil {
		return err
	}

	// 短暂延迟后再启动
	time.Sleep(100 * time.Millisecond)

	// 再启动
	return s.StartChannelInstance(id)
}

// GetChannelInstanceStatus 获取通道实例状态
func (s *ChannelInstanceServiceImpl) GetChannelInstanceStatus(id int64) (int, error) {
	err, instance := s.channelInstanceRepo.FindByID(id)
	if err != nil {
		return -1, err
	}
	return instance.Status, nil
}

// UpdateChannelInstanceHeartbeat 更新通道实例心跳
func (s *ChannelInstanceServiceImpl) UpdateChannelInstanceHeartbeat(id int64) error {
	// 获取通道实例
	err, instance := s.channelInstanceRepo.FindByID(id)
	if err != nil {
		return err
	}

	// 更新心跳时间
	instance.LastHeartbeat = time.Now()

	// 保存到数据库
	return s.channelInstanceRepo.Update(*instance)
}

// IncreaseConnectionCount 增加连接计数
func (s *ChannelInstanceServiceImpl) IncreaseConnectionCount(id int64) error {
	// 获取通道实例
	err, instance := s.channelInstanceRepo.FindByID(id)
	if err != nil {
		return err
	}

	// 检查是否超过最大连接数
	if instance.MaxConn > 0 && instance.CurrentConn >= instance.MaxConn {
		return errors.New("max connection limit reached")
	}

	// 更新连接计数
	return s.channelInstanceRepo.UpdateConnectionCount(id, true)
}

// DecreaseConnectionCount 减少连接计数
func (s *ChannelInstanceServiceImpl) DecreaseConnectionCount(id int64) error {
	return s.channelInstanceRepo.UpdateConnectionCount(id, false)
}

// validateChannelConfig 验证通道配置
func (s *ChannelInstanceServiceImpl) validateChannelConfig(channelType, config string) error {
	var values map[string]interface{}
	if err := json.Unmarshal([]byte(config), &values); err != nil {
		return errors.New("invalid channel config JSON: " + err.Error())
	}
	switch channelType {
	case data.ChannelTypeMQTT:
		url, _ := values["url"].(string)
		if strings.TrimSpace(url) == "" {
			host, _ := values["host"].(string)
			if strings.TrimSpace(host) == "" {
				return errors.New("MQTT config requires url or host")
			}
		}
	case data.ChannelTypeTCP, data.ChannelTypeTCPRaw, data.ChannelTypeTCPFrame, data.ChannelTypeModbusTCP:
		host, _ := values["host"].(string)
		if strings.TrimSpace(host) == "" {
			return errors.New(channelType + " config missing host")
		}
		if _, ok := values["port"]; !ok {
			return errors.New(channelType + " config missing port")
		}
	default:
		return errors.New("unsupported channel type: " + channelType)
	}

	return nil
}
