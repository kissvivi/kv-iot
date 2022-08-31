package data

var _ DeviceRepo = (*DeviceRepoImpl)(nil)

type DeviceRepoImpl struct {
}

func (d DeviceRepoImpl) Add() {
	panic("implement me")
}

func (d DeviceRepoImpl) Delete() {
	panic("implement me")
}

func (d DeviceRepoImpl) Update() {
	panic("implement me")
}

func (d DeviceRepoImpl) Select() {
	panic("implement me")
}

var _ PointRepo = (*PointRepoImpl)(nil)

type PointRepoImpl struct {
}

func (p PointRepoImpl) Add() {
	panic("implement me")
}

func (p PointRepoImpl) Delete() {
	panic("implement me")
}

func (p PointRepoImpl) Update() {
	panic("implement me")
}

func (p PointRepoImpl) Select() {
	panic("implement me")
}
