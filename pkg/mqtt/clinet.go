package mqtt

type mqttBroker interface {
	NewClient()
	Sub()
	Pub()
}

type Mqtt struct {
}
