package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/eip/miio2mqtt/miio"
	"gopkg.in/yaml.v2"
)

const (
	defaultPollInterval  = 10 * time.Second
	defaultPollAheadTime = 10 * time.Millisecond
	defaultPollTimeout   = 5 * time.Second
	defaultPushTimeout   = 4 * time.Second
	defaultMiioPort      = 54321
)

// Config defines application options
type Config struct {
	PollInterval  time.Duration               `yaml:"PollInterval"`
	PollAheadTime time.Duration               `yaml:"PollAheadTime"`
	PollTimeout   time.Duration               `yaml:"PollTimeout"`
	PushTimeout   time.Duration               `yaml:"PushTimeout"`
	Mqtt          MqttOptions                 `yaml:"MQTT"`
	MiioPort      int                         `yaml:"MiioPort"`
	Models        miio.Models                 `yaml:"Models"`
	Devices       map[string]miio.DeviceCfg   `yaml:"Devices"`
	Properties    map[interface{}]interface{} `yaml:"Properties"`
	Debug         bool                        `yaml:"Debug"`
	ChanStat      []int
}

type MqttOptions struct {
	BrokerURL string `yaml:"BrokerURL"`
}

func New() *Config {
	return &Config{
		PollInterval:  defaultPollInterval,
		PollAheadTime: defaultPollAheadTime,
		PollTimeout:   defaultPollTimeout,
		PushTimeout:   defaultPushTimeout,
		Mqtt:          MqttOptions{},
		MiioPort:      defaultMiioPort,
		Models:        miio.Models{"*": miio.DefaultModel()},
		Devices:       map[string]miio.DeviceCfg{},
		Properties: map[interface{}]interface{}{
			"off": 0,
			"on":  1,
		},
		Debug: false,
	}
}

// Load configuration from yaml file
func (c *Config) Load(path string) error {
	if len(path) == 0 {
		return errors.New("empty configuration file path")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	if err = c.parse(data); err != nil {
		return err
	}
	return c.validate()
}

func (c *Config) parse(data []byte) error {
	return yaml.Unmarshal(data, c)
}

func (c *Config) validate() error {
	for n, d := range c.Devices {
		token, err := hex.DecodeString(d.Token)
		if err != nil {
			return fmt.Errorf("invalid token %q for %s - %v", d.Token, n, err)
		}
		if len(token) != 16 {
			return fmt.Errorf("invalid token length %q for %s", d.Token, n)
		}
		// copy(d.Token[:], token)
		// d.Model = ""
		// c.Devices[n] = d
	}
	// yml, err := yaml.Marshal(C)
	// log.Print(string(yml), err)
	return nil
}

func (c *Config) UpdateChanStat(packets, updates int) {
	if c.ChanStat == nil {
		c.ChanStat = make([]int, 2)
	}
	if packets > c.ChanStat[0] {
		c.ChanStat[0] = packets
	}
	if updates > c.ChanStat[1] {
		c.ChanStat[1] = updates
	}
}
