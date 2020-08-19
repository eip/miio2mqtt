package net

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"testing"

	h "github.com/eip/miio2mqtt/helpers"
)

func Test_ParseUDPAddr(t *testing.T) {
	tests := []struct {
		host string
		port int
		want string
	}{
		{
			host: "0.0.0.0",
			port: 0,
			want: "0.0.0.0:0",
		},
		{
			host: "17.253.144.10",
			port: 123,
			want: "17.253.144.10:123",
		},
		{
			host: "255.255.255.255",
			port: 65535,
			want: "255.255.255.255:65535",
		},
		{
			host: "917.253.144.10",
			port: 123,
			want: "<nil>",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s:%d", tt.host, tt.port), func(t *testing.T) {
			got := ParseUDPAddr(tt.host, tt.port)
			h.AssertEqual(t, got.String(), tt.want)
		})
	}
}

func Test_GetLocalUDPAddr(t *testing.T) {
	tests := []struct {
		name string
		port int
		want string
		err  error
	}{
		{
			name: "LAN Address",
			port: 54321,
			want: `\d+\.\d+\.\d+\.\d+:54321`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetLocalUDPAddr(tt.port)
			h.AssertError(t, err, tt.err)
			matched, err := regexp.MatchString(tt.want, got.String())
			if err != nil {
				t.Errorf("\npattern error %#v", err)
				return
			}
			if !matched {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func Test_GetBroadcastUDPAddr(t *testing.T) {
	tests := []struct {
		name      string
		localAddr *net.UDPAddr
		port      int
		want      string
		err       error
	}{
		{
			name:      "17.253.144.255:321",
			localAddr: ParseUDPAddr("17.253.144.10", 321),
			port:      123,
			want:      `17\.253\.144\.255:123`,
		},
		{
			name: "LAN Broadcast Address",
			port: 54321,
			want: `\d+\.\d+\.\d+\.255:54321`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved := tt.localAddr.String()
			got, err := GetBroadcastUDPAddr(tt.localAddr, tt.port)
			h.AssertError(t, err, tt.err)
			matched, err := regexp.MatchString(tt.want, got.String()) // TODO h.AssertMatch()
			if err != nil {
				t.Fatalf("\npattern error %#v", err)
			}
			if !matched {
				t.Errorf("got %s, want %s", got, tt.want)
			}
			if tt.localAddr.String() != saved {
				t.Errorf("localAddr was %s, got %s", saved, tt.localAddr)
			}
		})
	}
}

func Test_IPv4ToInt(t *testing.T) {
	tests := []struct {
		name string
		want uint32
		err  error
	}{
		{
			name: "0.0.0.0",
			want: 0x00000000,
		},
		{
			name: "17.253.144.10",
			want: uint32(17)<<24 | uint32(253)<<16 | uint32(144)<<8 | uint32(10),
		},
		{
			name: "192.168.31.1",
			want: uint32(192)<<24 | uint32(168)<<16 | uint32(31)<<8 | uint32(1),
		},
		{
			name: "255.255.255.255",
			want: 0xffffffff,
		},
		{
			name: "555.255.255.255",
			err:  errors.New("invalid IPv4 address: <nil>"),
		},
		{
			name: "fe80::52ec:50ff:fe8c:3580",
			err:  errors.New("invalid IPv4 address: fe80::52ec:50ff:fe8c:3580"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IPv4ToInt(net.ParseIP(tt.name))
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_IPv4StrToInt(t *testing.T) {
	tests := []struct {
		name string
		want uint32
		err  error
	}{
		{
			name: "0.0.0.0",
			want: 0x00000000,
		},
		{
			name: "17.253.144.10",
			want: uint32(17)<<24 | uint32(253)<<16 | uint32(144)<<8 | uint32(10),
		},
		{
			name: "192.168.31.1",
			want: uint32(192)<<24 | uint32(168)<<16 | uint32(31)<<8 | uint32(1),
		},
		{
			name: "255.255.255.255",
			want: 0xffffffff,
		},
		{
			name: "555.255.255.255",
			err:  errors.New("invalid IPv4 address: 555.255.255.255"),
		},
		{
			name: "fe80::52ec:50ff:fe8c:3580",
			err:  errors.New("invalid IPv4 address: fe80::52ec:50ff:fe8c:3580"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IPv4StrToInt(tt.name)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}
