package mqtt

import (
	"fmt"
	"time"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"kv-iot/pkg/utils"
	"log"
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
	log.Printf("正在初始化MQTT客户端，连接到: %s:%d, 客户端ID: %s", broker, port, config.ClientID)
	
	opts := MQTT.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	
	// 设置客户端ID
	clientID := config.ClientID
	if clientID == "" {
		clientID = utils.NewGenID().String()
		log.Printf("使用自动生成的客户端ID: %s", clientID)
	}
	opts.SetClientID(clientID)
	
	// 设置认证信息
	if config.Username != "" {
		opts.SetUsername(config.Username)
		// 不记录密码到日志
	}
	if config.Password != "" {
		opts.SetPassword(config.Password)
	}
	
	// 设置自动重连参数
	opts.SetAutoReconnect(true)            // 启用自动重连
	opts.SetConnectRetryInterval(5 * time.Second)  // 重连间隔
	
	// 设置回调函数
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	
	// 创建客户端
	client := MQTT.NewClient(opts)
	
	// 连接到MQTT服务器
	log.Println("尝试连接到MQTT服务器...")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("MQTT连接失败: %v", token.Error())
		return nil
	}
	
	log.Println("MQTT客户端初始化成功")
	return &Mqtt{Conn: client, config: *config}
}

// connectHandler：连接的回调
var connectHandler MQTT.OnConnectHandler = func(client MQTT.Client) {
	reader := client.OptionsReader()
	log.Printf("client[%s]-Connected", reader.ClientID())
}

// connectLostHandler：连接丢失的回调
var connectLostHandler MQTT.ConnectionLostHandler = func(client MQTT.Client, err error) {
	reader := client.OptionsReader()
	log.Printf("client[%s]-Connect lost: %v", reader.ClientID(), err)
	// 由于启用了自动重连，这里不需要手动重连
}
