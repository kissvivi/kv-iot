package mqtt

import (
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

var messageSubHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())

}

func (m Mqtt) Sub(qos int, topic string) {
	token := m.Conn.Subscribe(topic, byte(qos), messageSubHandler)
	token.Wait()
	//fmt.Printf("Subscribed to topic %s\n", topic)
}
