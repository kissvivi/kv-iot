package mqtt

import (
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"kv-iot/pkg/utils"
)

type BrokerMqtt interface {
	Sub(qos int, topic string)
	Pub(qos int, topic string, text string)
}

type Mqtt struct {
	config ConfigMqtt
	Conn   MQTT.Client
}

type ConfigMqtt struct {
	Url      string
	Port     int
	ClientID string
	Username string
	Password string
}

func NewConfigMqtt(url string, port int, clientID string, username string, password string) *ConfigMqtt {
	return &ConfigMqtt{Url: url, Port: port, ClientID: clientID, Username: username, Password: password}
}

func NewMqtt(config *ConfigMqtt) *Mqtt {
	var broker = config.Url
	var port = config.Port
	opts := MQTT.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetClientID(config.ClientID)
	if config.ClientID == "" {
		opts.SetClientID(utils.NewGenID().String())
	}
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	return &Mqtt{Conn: client}
}

// connectHandler：连接的回调
var connectHandler MQTT.OnConnectHandler = func(client MQTT.Client) {
	reader := client.OptionsReader()
	fmt.Println(fmt.Sprintf("client[%s]-Connected", reader.ClientID()))
}

// connectLostHandler：连接丢失的回
var connectLostHandler MQTT.ConnectionLostHandler = func(client MQTT.Client, err error) {
	reader := client.OptionsReader()
	fmt.Printf("client[%s]-Connect lost: %v", reader.ClientID(), err)
}
