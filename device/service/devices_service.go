package service

import (
	"errors"
	"fmt"
	"kv-iot/db"
	"kv-iot/device/data"
	"kv-iot/device/data/repo"
	"strings"
	"time"

	"gorm.io/gorm"
)

var _ DevicesService = (*DevicesServiceImpl)(nil)

type DevicesService interface {
	AddDevices(devices data.Devices) (err error)
	UpdateDevices(devices data.Devices) (err error)
	DelDevices(devices data.Devices) (err error)
	GetDevices(devices data.Devices) (err error, deviceList []data.Devices)
	GetAllDevices() (err error, deviceList []data.Devices)
	GetDeviceByID(id int64) (*data.Devices, error)
	UpdateDevice(id int64, update DeviceUpdate) (*data.Devices, error)
}

type DeviceUpdate struct {
	Name              *string `json:"name"`
	DeviceNo          *string `json:"device_no"`
	ProductID         *int64  `json:"product_id"`
	ProductKey        *string `json:"product_key"`
	Desc              *string `json:"desc"`
	ProfileID         *int64  `json:"profile_id"`
	ChannelInstanceID *int64  `json:"channel_instance_id"`
	ParentDeviceID    *int64  `json:"parent_device_id"`
	Status            *int    `json:"status"`
	SubDevice         *int16  `json:"sub_device"`
	SubDeviceID       *int64  `json:"sub_device_id"`
	SubDeviceNo       *string `json:"sub_device_no"`
}

type DevicesServiceImpl struct {
	devices repo.DevicesRepo
}

func NewDevicesServiceImpl(devices repo.DevicesRepo) *DevicesServiceImpl {
	return &DevicesServiceImpl{devices: devices}
}

func (d DevicesServiceImpl) AddDevices(devices data.Devices) (err error) {
	now := time.Now()
	if devices.LastTime.IsZero() {
		devices.LastTime = now
	}
	if devices.LastSeenAt.IsZero() {
		devices.LastSeenAt = now
	}
	if err := d.validateDevice(&devices); err != nil {
		return err
	}
	return d.devices.Add(devices)
}

func (d DevicesServiceImpl) UpdateDevices(devices data.Devices) (err error) {
	if err := d.validateDevice(&devices); err != nil {
		return err
	}
	return d.devices.Update(devices)
}

func (d DevicesServiceImpl) DelDevices(devices data.Devices) (err error) {
	return d.devices.Delete(devices)
}

func (d DevicesServiceImpl) GetAllDevices() (err error, deviceList []data.Devices) {
	err, deviceList = d.devices.FindAll()
	return
}

func (d DevicesServiceImpl) GetDevices(devices data.Devices) (err error, deviceList []data.Devices) {
	err, deviceList = d.devices.FindByStruct(devices)
	return
}

func (d DevicesServiceImpl) GetDeviceByID(id int64) (*data.Devices, error) {
	err, devices := d.GetDevices(data.Devices{Model: gorm.Model{ID: uint(id)}})
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &devices[0], nil
}

func (d DevicesServiceImpl) UpdateDevice(id int64, update DeviceUpdate) (*data.Devices, error) {
	device, err := d.GetDeviceByID(id)
	if err != nil {
		return nil, err
	}

	applyDeviceUpdate(device, update)
	if update.ProductID != nil && update.ProductKey == nil {
		device.ProductKey = ""
	}
	if err := d.validateDevice(device); err != nil {
		return nil, err
	}
	if err := db.MYSQLDB.Save(device).Error; err != nil {
		return nil, err
	}
	return device, nil
}

func (d DevicesServiceImpl) validateDevice(device *data.Devices) error {
	device.Name = strings.TrimSpace(device.Name)
	device.DeviceNo = strings.TrimSpace(device.DeviceNo)
	device.ProductKey = strings.TrimSpace(device.ProductKey)

	if device.Name == "" {
		return errors.New("device name is required")
	}
	if device.DeviceNo == "" {
		return errors.New("device_no is required")
	}
	if device.ProductID <= 0 {
		return errors.New("product_id is required")
	}

	err, product := (&repo.ProductsRepo{}).FindByStruct(data.Products{
		Model: gorm.Model{ID: uint(device.ProductID)},
	})
	if err != nil {
		return err
	}
	if len(product) == 0 {
		return fmt.Errorf("product %d not found", device.ProductID)
	}
	if device.ProductKey == "" {
		device.ProductKey = product[0].ProductKey
	} else if device.ProductKey != product[0].ProductKey {
		return errors.New("product_key does not match product_id")
	}

	err, sameNoDevices := d.GetDevices(data.Devices{DeviceNo: device.DeviceNo})
	if err != nil {
		return err
	}
	for _, existing := range sameNoDevices {
		if existing.ID != device.ID {
			return errors.New("device_no already exists")
		}
	}

	if device.ProfileID > 0 {
		profileRepo := repo.NewProductProtocolProfileRepo(db.MYSQLDB)
		err, profile := profileRepo.FindByID(device.ProfileID)
		if err != nil {
			return err
		}
		if profile == nil {
			return fmt.Errorf("protocol profile %d not found", device.ProfileID)
		}
		if profile.ProductID != device.ProductID {
			return errors.New("profile_id does not belong to product_id")
		}
	}

	if device.ChannelInstanceID > 0 {
		channelRepo := repo.NewChannelInstanceRepo(db.MYSQLDB)
		err, channel := channelRepo.FindByID(device.ChannelInstanceID)
		if err != nil {
			return err
		}
		if channel == nil {
			return fmt.Errorf("channel instance %d not found", device.ChannelInstanceID)
		}
	}

	if device.ParentDeviceID > 0 {
		if device.ParentDeviceID == int64(device.ID) {
			return errors.New("device cannot be its own parent")
		}
		if _, err := d.GetDeviceByID(device.ParentDeviceID); err != nil {
			return fmt.Errorf("parent device not found: %w", err)
		}
	}

	if device.LastTime.IsZero() {
		device.LastTime = time.Now()
	}
	return nil
}

func applyDeviceUpdate(device *data.Devices, update DeviceUpdate) {
	if update.Name != nil {
		device.Name = *update.Name
	}
	if update.DeviceNo != nil {
		device.DeviceNo = *update.DeviceNo
	}
	if update.ProductID != nil {
		device.ProductID = *update.ProductID
	}
	if update.ProductKey != nil {
		device.ProductKey = *update.ProductKey
	}
	if update.Desc != nil {
		device.Desc = *update.Desc
	}
	if update.ProfileID != nil {
		device.ProfileID = *update.ProfileID
	}
	if update.ChannelInstanceID != nil {
		device.ChannelInstanceID = *update.ChannelInstanceID
	}
	if update.ParentDeviceID != nil {
		device.ParentDeviceID = *update.ParentDeviceID
	}
	if update.Status != nil {
		device.Status = *update.Status
	}
	if update.SubDevice != nil {
		device.SubDevice = *update.SubDevice
	}
	if update.SubDeviceID != nil {
		device.SubDeviceID = *update.SubDeviceID
	}
	if update.SubDeviceNo != nil {
		device.SubDeviceNo = *update.SubDeviceNo
	}
}
