package mqtt

import (
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func (m Mqtt) Pub(qos int, topic string, text string) {
	token := m.Conn.Publish(topic, byte(qos), false, text)
	token.Wait()
	fmt.Printf("Pub to topic %s", topic)
}

// 全局pub消息处理
var messagePubHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}
