package mqtt

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/eip/miio2mqtt/config"
	"github.com/eip/miio2mqtt/miio"
	log "github.com/go-pkgz/lgr"
)

type Message struct {
	Device *miio.Device
}

type Client struct {
	mqtt mqtt.Client
}

func NewClient() *Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.C.Mqtt.BrokerURL)
	opts.SetClientID(fmt.Sprintf("miio2mqtt-%x", time.Now().UnixNano()%0x1000000))
	opts.SetConnectTimeout(config.C.PushTimeout)
	opts.SetAutoReconnect(false)
	opts.SetConnectionLostHandler(connectionLostHandler)
	return &Client{mqtt.NewClient(opts)}
}

func (c *Client) Connect() error {
	if c.mqtt.IsConnected() {
		return nil
	}
	log.Printf("[DEBUG] connecting to %v...", config.C.Mqtt.BrokerURL)
	token := c.mqtt.Connect()
	if token.Wait() && token.Error() != nil {
		err := token.Error()
		return err
	}
	log.Printf("[DEBUG] connected to %v", config.C.Mqtt.BrokerURL)
	return nil
}

func (c *Client) Disconnect() {
	c.mqtt.Disconnect(uint(config.C.PushTimeout / time.Millisecond))
	log.Printf("[DEBUG] disconnected from %v", config.C.Mqtt.BrokerURL)
}

func (c *Client) Publish(message Message) error {
	if err := c.Connect(); err != nil {
		return err
	}
	if token := c.mqtt.Publish(message.Device.Topic, 0, true, message.Device.Properties); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	message.Device.SetStatePublishedNow()
	log.Printf("[DEBUG] publish to %s: %s", message.Device.Topic, message.Device.Properties)
	return nil
}

func connectionLostHandler(client mqtt.Client, err error) {
	log.Printf("[WARN] disconnected from %v: %v", config.C.Mqtt.BrokerURL, err)
}
