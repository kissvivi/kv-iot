package mqttchan

import (
	"kv-iot/datacenter/data"
	"sync"
)

// DeviceAuths 设备认证 map
type DeviceAuths struct {
	Items sync.Map
}

func NewDeviceAuthsMap() *DeviceAuths {
	return &DeviceAuths{
		Items: sync.Map{},
	}
}

func (dv *DeviceAuths) Set(device data.Device) {
	dv.Items.Store(device.DeviceNo, device)
}

func (dv *DeviceAuths) Get(deviceNo int) *data.Device {
	if vehicle, ok := dv.Items.Load(deviceNo); ok {
		return vehicle.(*data.Device)
	}
	return nil
}

func (dv *DeviceAuths) GetAll() []*data.Device {
	var vs []*data.Device
	dv.Items.Range(func(deviceNo, device interface{}) bool {
		vs = append(vs, device.(*data.Device))
		return true
	})
	return vs
}

func (dv *DeviceAuths) Delete(deviceNo interface{}) {
	dv.Items.Delete(deviceNo)
}

func (dv *DeviceAuths) Has(deviceNo interface{}) bool {
	_, ok := dv.Items.Load(deviceNo)
	return ok
}
