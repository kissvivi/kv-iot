package publicdevice

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("device not found")

type Catalog interface {
	LoadMobileDevice(ctx context.Context, deviceID int64) (Device, error)
}

type Device struct {
	ID          uint
	Name        string
	DeviceNo    string
	Description string
	ProductID   int64
	ProductKey  string
	ProductName string
	Status      int
	LastSeenAt  time.Time
	Properties  []Property
	Actions     []Action
}

type Property struct {
	ID           uint
	Identifier   string
	Name         string
	DataType     string
	Unit         string
	ReadWrite    string
	Precision    int
	MinValue     *float64
	MaxValue     *float64
	EnumValues   string
	DefaultValue string
}

type Action struct {
	ID               uint
	Identifier       string
	Name             string
	InputSchema      string
	TimeoutMs        int
	ReadbackRequired bool
	Available        bool
}
