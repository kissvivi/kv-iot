package mqtt

import (
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// 订阅来的消息
var _SubData = make(chan []byte, 100)
var messageSubHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
	_SubData <- msg.Payload()
}

func (m Mqtt) Sub(qos int, topic string) {
	token := m.Conn.Subscribe(topic, byte(qos), messageSubHandler)
	token.Wait()
	//fmt.Printf("Subscribed to topic %s\n", topic)
}
