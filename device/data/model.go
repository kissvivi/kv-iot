package data

// Device 设备信息
type Device struct {
}

// Point 点信息
type Point struct {
}

type DeviceRepo interface {
	Add()
	Delete()
	Update()
	Select()
}

type PointRepo interface {
	Add()
	Delete()
	Update()
	Select()
}
