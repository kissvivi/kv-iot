package data

type DeviceChanConfig struct {
	Ip       string
	Port     int
	ClientID string
	UserName string
	Password string
	Secret   string
}

func (d DeviceChanConfig) genDeviceChanSecret() string {
	return d.ClientID
}

//func genDeviceSecret()  string{
//
//}
