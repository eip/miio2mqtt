package mqtt

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/eip/miio2mqtt/config"
	log "github.com/go-pkgz/lgr"
)

type Message struct {
	Topic   string
	Payload string
}

func NewClient() mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.C.Mqtt.BrokerURL)
	opts.SetClientID(fmt.Sprintf("miio2mqtt-%x", time.Now().UnixNano()%0x1000000))
	opts.SetConnectTimeout(config.C.PushTimeout)
	// opts.SetKeepAlive(60 * time.Second)
	opts.SetAutoReconnect(false)
	// opts.SetMaxReconnectInterval(5 * time.Second)

	// opts.SetDefaultPublishHandler(w.getMessageHandler())
	// opts.SetOnConnectHandler(w.getConnectHandler())
	opts.SetConnectionLostHandler(connectionLostHandler)

	return mqtt.NewClient(opts)
}

func connectionLostHandler(client mqtt.Client, err error) {
	log.Printf("[WARN] disconnected from %v: %v", config.C.Mqtt.BrokerURL, err)
}

func Connect(client mqtt.Client) error {
	if client.IsConnected() {
		return nil
	}
	log.Printf("[DEBUG] connecting to %v...", config.C.Mqtt.BrokerURL)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		err := token.Error()
		return err
	}
	log.Printf("[DEBUG] connected to %v", config.C.Mqtt.BrokerURL)
	return nil
}

func Publish(client mqtt.Client, message Message) error {
	if err := Connect(client); err != nil {
		return err
	}
	if token := client.Publish(message.Topic, 0, true, message.Payload); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	log.Printf("[INFO] MQTT publish %s: %s", message.Topic, message.Payload)
	return nil
}
