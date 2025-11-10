package data

import (
	"gorm.io/gorm"
	"time"
)

type Devices struct {
	//ID          int64     `json:"id" gorm:"column:id"`
	gorm.Model
	Name        string    `json:"name" gorm:"column:name"` // 设备名称
	DeviceNo    string    `json:"device_no" gorm:"column:device_no"`
	ProductID   int64     `json:"product_id" gorm:"column:product_id"`       // 所属产品id
	Desc        string    `json:"desc" gorm:"column:desc"`                   // 设备描述
	LastTime    time.Time `json:"last_time" gorm:"column:last_time"`         // 最后在线时间
	SubDevice   int16     `json:"sub_device" gorm:"column:sub_device"`       // 是否子设备
	SubDeviceID int64     `json:"sub_device_id" gorm:"column:sub_device_id"` // 子设备id
	SubDeviceNo string    `json:"sub_device_no" gorm:"column:sub_device_no"` // 子设备号
	ProductKey  string    `json:"product_key" gorm:"column:product_key"`     // 产品标识
}

func (m *Devices) TableName() string {
	return "devices"
}

type DevicesRepo interface {
	Add(devices Devices) error
	Delete(devices Devices) error
	FindAll() (error, []Devices)
	FindByStruct(devices Devices) (error, []Devices)
}
