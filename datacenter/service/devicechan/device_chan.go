package devicechan

type IDeviceChan interface {
	Create()  //创建通道
	Disable() //禁用通道
	Close()   //关闭通道
}
