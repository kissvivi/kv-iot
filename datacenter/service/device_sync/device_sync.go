package device_sync

import (
	"fmt"
	"kv-iot/config"
	"kv-iot/datacenter/service/device_auth"
	"log"
	"time"
)

// DeviceSyncService 设备同步服务接口
type DeviceSyncService interface {
	SyncDevices() error              // 同步所有设备
	SyncDevice(productKey, deviceNo string) error // 同步单个设备
	StartSyncScheduler(interval time.Duration)    // 启动定时同步
	StopSyncScheduler()            // 停止定时同步
}

// DeviceSyncServiceImpl 设备同步服务实现
type DeviceSyncServiceImpl struct {
	authMap      *device_auth.DeviceAuthsMap
	deviceClient DeviceCenterClient
	running      bool
	ticker       *time.Ticker
	stopCh       chan bool
	cfg          *config.Config
}

// NewDeviceSyncService 创建设备同步服务实例
func NewDeviceSyncService() *DeviceSyncServiceImpl {
	// 初始化配置
	cfg, err := config.InitConfig()
	if err != nil {
		log.Printf("初始化配置失败: %v", err)
		// 使用默认配置结构体
		cfg = &config.Config{}
	}
	
	// 创建设备中心客户端
	deviceClient := NewDeviceCenterClient("mysql")
	
	return &DeviceSyncServiceImpl{
		authMap:      device_auth.NewDeviceAuthsMap(),
		deviceClient: deviceClient,
		running:      false,
		stopCh:       make(chan bool, 1),
		cfg:          cfg,
	}
}

// SyncDevices 同步所有设备
func (s *DeviceSyncServiceImpl) SyncDevices() error {
	log.Println("开始同步设备信息到认证系统...")
	
	// 使用设备中心客户端获取真实设备信息并同步到认证映射表
	err := SyncDevicesToAuthMap(s.authMap, s.deviceClient)
	if err != nil {
		log.Printf("同步设备失败: %v", err)
		return fmt.Errorf("设备同步失败: %w", err)
	}
	
	log.Printf("设备同步完成，共同步 %d 台设备到认证系统", s.authMap.Size())
	return nil
}

// SyncDevice 同步单个设备
func (s *DeviceSyncServiceImpl) SyncDevice(productKey, deviceNo string) error {
	deviceUserName := fmt.Sprintf("%s:%s", productKey, deviceNo)
	
	// 使用设备中心客户端查询单个设备
	device, err := s.deviceClient.GetDeviceByKeyAndNo(productKey, deviceNo)
	if err != nil {
		log.Printf("查询设备失败: %v", err)
		return fmt.Errorf("查询设备失败: %w", err)
	}
	
	// 添加到认证映射表
	s.authMap.Add(deviceUserName)
	log.Printf("已同步单个设备: %s (%s)", deviceUserName, device.Name)
	return nil
}

// StartSyncScheduler 启动定时同步
func (s *DeviceSyncServiceImpl) StartSyncScheduler(interval time.Duration) {
	if s.running {
		log.Println("设备同步调度器已经在运行中")
		return
	}
	
	s.running = true
	s.ticker = time.NewTicker(interval)
	
	go func() {
		log.Printf("设备同步调度器已启动，同步间隔: %v", interval)
		
		// 立即执行一次同步
		if err := s.SyncDevices(); err != nil {
			log.Printf("初始设备同步失败: %v", err)
		}
		
		for {
			select {
			case <-s.ticker.C:
				if err := s.SyncDevices(); err != nil {
					log.Printf("定时设备同步失败: %v", err)
				}
			case <-s.stopCh:
				s.ticker.Stop()
				s.running = false
				log.Println("设备同步调度器已停止")
				return
			}
		}
	}()
}

// StopSyncScheduler 停止定时同步
func (s *DeviceSyncServiceImpl) StopSyncScheduler() {
	if !s.running {
		log.Println("设备同步调度器未在运行")
		return
	}
	
	s.stopCh <- true
}