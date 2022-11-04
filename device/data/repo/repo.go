package repo

import "github.kissvivi.kv-iot/db"

type baseRepo interface {
}

var _ baseRepo = (*DeviceRepoImpl)(nil)

type DeviceRepoImpl struct {
	DB db.BaseDB
}

func (d DeviceRepoImpl) Add(in interface{}) (out interface{}) {
	panic("implement me")
}

func (d DeviceRepoImpl) Delete(in interface{}) (out interface{}) {
	panic("implement me")
}

func (d DeviceRepoImpl) Update(in interface{}) (out interface{}) {
	panic("implement me")
}

func (d DeviceRepoImpl) Select(in interface{}) (out interface{}) {
	panic("implement me")
}
