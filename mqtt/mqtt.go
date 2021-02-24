package mqtt

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/eip/miio2mqtt/config"
	h "github.com/eip/miio2mqtt/helpers"
	"github.com/eip/miio2mqtt/miio"
	log "github.com/go-pkgz/lgr"
)

type Client struct {
	mqtt   mqtt.Client
	config *config.Config
}

func NewClient(config *config.Config) *Client {
	client := &Client{config: config}
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Mqtt.BrokerURL)
	opts.SetClientID(fmt.Sprintf("miio2mqtt-%x", time.Now().UnixNano()%0x1000000))
	opts.SetConnectTimeout(config.PushTimeout)
	opts.SetAutoReconnect(false)
	opts.SetConnectionLostHandler(client.connectionLostHandler())
	client.mqtt = mqtt.NewClient(opts)
	return client
}

func (c *Client) Connect() error {
	if c.mqtt.IsConnected() {
		return nil
	}
	log.Printf("[DEBUG] connecting to %v...", c.config.Mqtt.BrokerURL)
	token := c.mqtt.Connect()
	if token.Wait() && token.Error() != nil {
		err := token.Error()
		return err
	}
	log.Printf("[DEBUG] connected to %v", c.config.Mqtt.BrokerURL)
	return nil
}

func (c *Client) Disconnect() {
	c.mqtt.Disconnect(uint(c.config.PushTimeout / time.Millisecond))
	log.Printf("[DEBUG] disconnected from %v", c.config.Mqtt.BrokerURL)
}

func (c *Client) Publish(device *miio.Device) error {
	if err := c.Connect(); err != nil {
		return err
	}
	if token := c.mqtt.Publish(device.Topic, 0, true, device.Properties()); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	device.SetStatePublishedNow()
	log.Printf("[DEBUG] publish to %s: %s", device.Topic, h.StripJSONQuotes(device.Properties()))
	return nil
}

func (c *Client) connectionLostHandler() mqtt.ConnectionLostHandler {
	return func(client mqtt.Client, err error) {
		log.Printf("[WARN] disconnected from %v: %v", c.config.Mqtt.BrokerURL, err)
	}
}
