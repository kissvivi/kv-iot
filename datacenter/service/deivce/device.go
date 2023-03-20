package deivce

import "kv-iot/datacenter/data"

type IDevice interface {
	RegDevice()
	AuthDevice(msg data.KvMsg) (err error)
	StateDevice(msg data.KvMsg) (err error)
	SendMsg()
}
