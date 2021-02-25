package mqtt

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/eip/miio2mqtt/config"
	h "github.com/eip/miio2mqtt/helpers"
	"github.com/eip/miio2mqtt/miio"
)

var testLog = h.InitTestLog()

type mockMqttToken struct {
	err error
}

func (t *mockMqttToken) Wait() bool {
	return true
}

func (t *mockMqttToken) WaitTimeout(time.Duration) bool {
	return true
}

func (t *mockMqttToken) Done() <-chan struct{} {
	return nil
}

func (t *mockMqttToken) Error() error {
	return t.err
}

type mockMqttClient struct {
	opts            *mqtt.ClientOptions
	isConnected     bool
	connectErr      error
	publishErr      error
	publishData     string
	connectCalls    int
	disconnectCalls int
	publishCalls    int
}

func mockMqttClientFactory(opts *mqtt.ClientOptions) mqtt.Client {
	return &mockMqttClient{opts: opts}
}

func (c *mockMqttClient) AddRoute(topic string, callback mqtt.MessageHandler) {}

func (c *mockMqttClient) IsConnected() bool {
	return c.isConnected
}

func (c *mockMqttClient) IsConnectionOpen() bool {
	return false
}

func (c *mockMqttClient) Connect() mqtt.Token {
	c.connectCalls++
	return &mockMqttToken{err: c.connectErr}
}

func (c *mockMqttClient) Disconnect(quiesce uint) {
	c.disconnectCalls++
}

func (c *mockMqttClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	c.publishCalls++
	c.publishData = fmt.Sprintf("%s: %s", topic, payload)
	return &mockMqttToken{err: c.publishErr}
}

func (c *mockMqttClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	return nil
}

func (c *mockMqttClient) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	return nil
}

func (c *mockMqttClient) Unsubscribe(topics ...string) mqtt.Token {
	return nil
}

func (c *mockMqttClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.ClientOptionsReader{}
}

func (c *mockMqttClient) UpdateLastReceived() {}

func (c *mockMqttClient) UpdateLastSent() {}

func init() {
	mqttFactory = mockMqttClientFactory
}

func testConfig() *config.Config {
	config := config.New()
	config.Mqtt.BrokerURL = "tcp://localhost:1883"
	return config
}

func testDevice(topic, properties string) *miio.Device {
	device := &miio.Device{
		DeviceCfg: miio.DeviceCfg{Topic: topic},
	}
	device.SetProperties(properties)
	return device
}

func Test_NewClient(t *testing.T) {
	config := testConfig()
	got := NewClient(config)
	h.AssertEqual(t, got.config, config)
	gotOpts := got.mqtt.(*mockMqttClient).opts
	h.AssertEqual(t, len(gotOpts.Servers), 1)
	h.AssertEqual(t, gotOpts.ClientID, regexp.MustCompile(`miio2mqtt-[0-9a-f]{6}$`))
}

func TestClient_createOptions(t *testing.T) {
	config := testConfig()
	client := &Client{config: config}
	got := client.createOptions()
	brokerURL, _ := url.Parse(config.Mqtt.BrokerURL)
	h.AssertEqual(t, got.Servers, []*url.URL{brokerURL})
	h.AssertEqual(t, got.ClientID, regexp.MustCompile(`miio2mqtt-[0-9a-f]{6}$`))
	h.AssertEqual(t, got.ConnectTimeout, config.PushTimeout)
	h.AssertEqual(t, got.AutoReconnect, false)

	testLog.Reset()
	got.OnConnectionLost(nil, errors.New("connection lost error"))
	h.AssertEqual(t, testLog.Message, "[WARN]  disconnected from tcp://localhost:1883: connection lost error\n")

	def := mqtt.NewClientOptions()
	got.Servers = def.Servers
	got.ClientID = def.ClientID
	got.ConnectTimeout = def.ConnectTimeout
	got.AutoReconnect = def.AutoReconnect
	got.OnConnectionLost, def.OnConnectionLost = nil, nil // reflect.DeepEqual func values workaround
	h.AssertEqual(t, got, def)
}

func TestClient_Connect(t *testing.T) {
	tests := []struct {
		name         string
		client       *Client
		err          error
		connectCalls int
	}{
		{
			name:         "Already connected",
			client:       func() *Client { c := NewClient(testConfig()); c.mqtt.(*mockMqttClient).isConnected = true; return c }(),
			err:          nil,
			connectCalls: 0,
		},
		{
			name:         "Success",
			client:       NewClient(testConfig()),
			err:          nil,
			connectCalls: 1,
		},
		{
			name: "Error",
			client: func() *Client {
				c := NewClient(testConfig())
				c.mqtt.(*mockMqttClient).connectErr = errors.New("connect error")
				return c
			}(),
			err:          errors.New("connect error"),
			connectCalls: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.client.mqtt.(*mockMqttClient)
			h.AssertEqual(t, mock.connectCalls, 0)
			err := tt.client.Connect()
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, mock.connectCalls, tt.connectCalls)
		})
	}
}

func TestClient_Disconnect(t *testing.T) {
	client := NewClient(testConfig())
	mock := client.mqtt.(*mockMqttClient)
	h.AssertEqual(t, mock.disconnectCalls, 0)
	client.Disconnect()
	h.AssertEqual(t, mock.disconnectCalls, 1)
}

func TestClient_Publish(t *testing.T) {
	tests := []struct {
		name         string
		client       *Client
		arg          *miio.Device
		err          error
		connectCalls int
		publishCalls int
		publishData  string
	}{
		{
			name: "Connect error",
			client: func() *Client {
				c := NewClient(testConfig())
				c.mqtt.(*mockMqttClient).connectErr = errors.New("connect error")
				return c
			}(),
			arg:          testDevice("home/devices/test", "test properties"),
			err:          errors.New("connect error"),
			connectCalls: 1,
			publishCalls: 0,
		},
		{
			name: "Publish error",
			client: func() *Client {
				c := NewClient(testConfig())
				c.mqtt.(*mockMqttClient).publishErr = errors.New("publish error")
				return c
			}(),
			arg:          testDevice("home/devices/test", "test properties"),
			err:          errors.New("publish error"),
			connectCalls: 1,
			publishCalls: 1,
			publishData:  "home/devices/test: test properties",
		},
		{
			name:         "Success",
			client:       NewClient(testConfig()),
			arg:          testDevice("home/devices/test", "test properties"),
			connectCalls: 1,
			publishCalls: 1,
			publishData:  "home/devices/test: test properties",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.client.mqtt.(*mockMqttClient)
			h.AssertEqual(t, mock.connectCalls, 0)
			h.AssertEqual(t, mock.publishCalls, 0)
			err := tt.client.Publish(tt.arg)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, mock.connectCalls, tt.connectCalls)
			h.AssertEqual(t, mock.publishCalls, tt.publishCalls)
			h.AssertEqual(t, mock.publishData, tt.publishData)
		})
	}
}
