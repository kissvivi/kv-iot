package deivce

type IDevice interface {
	RegDevice()
	AuthDevice()
	StateDevice()
	SendMsg()
}
