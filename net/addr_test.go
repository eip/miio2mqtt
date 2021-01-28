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

func Test_GetUDPAddresses(t *testing.T) {
	tests := []struct {
		name      string
		port      int
		wantLocal *regexp.Regexp
		wantBC    *regexp.Regexp
		err       error
	}{
		{
			name:      "LAN Address",
			port:      54321,
			wantLocal: regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.){3}(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9]):0$`),
			wantBC:    regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.){3}(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9]):54321$`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLocal, gotBC, err := GetUDPAddresses(tt.port)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, gotLocal, tt.wantLocal)
			h.AssertEqual(t, gotBC, tt.wantBC)
		})
	}
}

func Test_getLocalIPAddr(t *testing.T) {
	tests := []struct {
		name string
		want *regexp.Regexp
		err  error
	}{
		{
			name: "LAN Address",
			want: regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.){3}(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])$`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getLocalIPAddr()
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_getLocalIPNet(t *testing.T) {
	tests := []struct {
		name      string
		localAddr *net.IP
		want      *regexp.Regexp
		err       error
	}{
		{
			name:      "LAN Address",
			localAddr: func() *net.IP { a, _ := getLocalIPAddr(); return a }(),
			want:      regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.){3}(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])/(?:3[0-1]|[1-2][0-9]|[8-9])$`),
		},
		{
			name:      "Loopback Address",
			localAddr: func() *net.IP { a := net.ParseIP("127.0.0.1"); return &a }(),
			want:      regexp.MustCompile("<nil>"),
			err:       errors.New("cannot find interface with IP address 127.0.0.1"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getLocalIPNet(tt.localAddr)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_getBroadcastIPAddr(t *testing.T) {
	tests := []struct {
		name       string
		localIPNet *net.IPNet
		want       *regexp.Regexp
		err        error
	}{
		{
			name:       "LAN Address",
			localIPNet: func() *net.IPNet { a, _ := getLocalIPAddr(); n, _ := getLocalIPNet(a); return n }(),
			want:       regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.){3}(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])$`),
		},
		{
			name:       "192.168.31.1/24",
			localIPNet: &net.IPNet{IP: net.ParseIP("192.168.31.1"), Mask: net.CIDRMask(24, 32)},
			want:       regexp.MustCompile("192\\.168\\.31\\.255"),
		},
		{
			name:       "10.0.2.13/24",
			localIPNet: &net.IPNet{IP: net.ParseIP("10.0.2.13"), Mask: net.CIDRMask(8, 32)},
			want:       regexp.MustCompile("10\\.255\\.255\\.255"),
		},
		{
			name:       "17.253.144.10/21",
			localIPNet: &net.IPNet{IP: net.ParseIP("17.253.144.10"), Mask: net.CIDRMask(21, 32)},
			want:       regexp.MustCompile("17\\.253\\.151\\.255"),
		},
		{
			name:       "2001:db8:abcd:3f00::/64",
			localIPNet: &net.IPNet{IP: net.ParseIP("2001:db8:abcd:3f00::"), Mask: net.CIDRMask(64, 128)},
			want:       regexp.MustCompile("<nil>"),
			err:        errors.New("invalid IPv4 address: 2001:db8:abcd:3f00::/64"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getBroadcastIPAddr(tt.localIPNet)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
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
