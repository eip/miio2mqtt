package config

import (
	"errors"
	"testing"
	"time"

	h "github.com/eip/miio2mqtt/helpers"
	"github.com/eip/miio2mqtt/miio"
)

func Test_New(t *testing.T) {
	tests := []struct {
		name string
		want *Config
	}{
		{
			name: "Default",
			want: &Config{
				PollInterval:  defaultPollInterval,
				PollAheadTime: defaultPollAheadTime,
				PollTimeout:   defaultPollTimeout,
				PushTimeout:   defaultPushTimeout,
				MiioPort:      defaultMiioPort,
				Models: miio.Models{
					"*": miio.Model{
						Methods: miio.ModelMethods{
							MiioInfo: defaultMiioInfoRequest,
							GetProp:  defaultGetPropRequest,
						},
						Params: defaultGetPropParams,
					},
				},
				Devices: map[string]miio.DeviceCfg{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New()
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_Config_parse(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want *Config
		err  error
	}{
		{
			name: "Default",
			want: New(),
		},
		{
			name: "Sample",
			data: []byte(`PollInterval: 10s
PollAheadTime: 50ms
PollTimeout: 5s
PushTimeout: 4s
MiioPort: 12345
Models:
  mi.dummy.v1:
    Methods:
      MiioInfo: '{"dummy":"mi.dummy.v1","method":"miIO.info","params":[],"id":#}'
      GetProp: '{"dummy":"mi.dummy.v1","method":"get_prop","params":#,"id":#}'
    Params:
      - power
      - state
      - battery
      - time
Devices:
  DummySensor:
    Address: 192.168.1.200
    ID: 0x01234567
    Token: 0102030405060708090a0b0c0d0e0f10
    Topic: home/room/dummysensor`),
			want: &Config{
				PollInterval:  10 * time.Second,
				PollAheadTime: 50 * time.Millisecond,
				PollTimeout:   5 * time.Second,
				PushTimeout:   4 * time.Second,
				MiioPort:      12345,
				Models: miio.Models{
					"*": miio.Model{
						Methods: miio.ModelMethods{
							MiioInfo: defaultMiioInfoRequest,
							GetProp:  defaultGetPropRequest,
						},
						Params: defaultGetPropParams,
					},
					"mi.dummy.v1": miio.Model{
						Methods: miio.ModelMethods{
							MiioInfo: `{"dummy":"mi.dummy.v1","method":"miIO.info","params":[],"id":#}`,
							GetProp:  `{"dummy":"mi.dummy.v1","method":"get_prop","params":#,"id":#}`,
						},
						Params: []string{"power", "state", "battery", "time"},
					},
				},
				Devices: map[string]miio.DeviceCfg{
					"DummySensor": {
						Address: "192.168.1.200",
						ID:      0x01234567,
						Topic:   "home/room/dummysensor",
						Token:   "0102030405060708090a0b0c0d0e0f10",
					},
				},
			},
		},
		{
			name: "Invalid 1",
			data: []byte(`PollInterval: foo`),
			want: New(),
			err:  errors.New("yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `foo` into time.Duration"),
		},
		{
			name: "Invalid 2",
			data: []byte(`Not a yaml data`),
			want: New(),
			err:  errors.New("yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `Not a y...` into config.Config"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := New()
			err := config.parse(tt.data)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, config, tt.want)
		})
	}
}

func Test_Config_validate(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   *Config
		err    error
	}{
		{
			name:   "Default",
			config: New(),
			want:   New(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, tt.config, tt.want)
		})
	}
}
