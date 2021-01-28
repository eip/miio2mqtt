package net

import (
	"encoding/binary"
	"fmt"
	"net"
)

const probeAddress = "1.1.1.1:53"

func ParseUDPAddr(host string, port int) *net.UDPAddr {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	udpAddr, err := net.ResolveUDPAddr(udpNetwork, addr)
	if err != nil {
		return nil
	}
	udpAddr.IP = udpAddr.IP.To4()
	return udpAddr
}

func GetUDPAddresses(port int) (*net.UDPAddr, *net.UDPAddr, error) {
	localAddr, err := getLocalIPAddr()
	if err != nil {
		return nil, nil, err
	}
	localIPNet, err := getLocalIPNet(localAddr)
	if err != nil {
		return nil, nil, err
	}
	bcastAddr, err := getBroadcastIPAddr(localIPNet)
	if err != nil {
		return nil, nil, err
	}
	return &net.UDPAddr{IP: *localAddr, Port: 0, Zone: ""}, &net.UDPAddr{IP: *bcastAddr, Port: port, Zone: ""}, nil
}

func getLocalIPAddr() (*net.IP, error) {
	conn, err := net.Dial(udpNetwork, probeAddress)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	addr := conn.LocalAddr().(*net.UDPAddr).IP
	return &addr, nil
}

func getLocalIPNet(localAddr *net.IP) (*net.IPNet, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil && !ipNet.IP.IsLoopback() && ipNet.IP.String() == localAddr.String() {
			return ipNet, nil
		}
	}
	return nil, fmt.Errorf("cannot find interface with IP address %s", localAddr)
}

func getBroadcastIPAddr(localIPNet *net.IPNet) (*net.IP, error) {
	if ip, mask := localIPNet.IP.To4(), net.IP(localIPNet.Mask).To4(); ip != nil && mask != nil {
		addr := make(net.IP, net.IPv4len)
		binary.BigEndian.PutUint32(addr, binary.BigEndian.Uint32(ip)|^binary.BigEndian.Uint32(mask))
		return &addr, nil
	}
	return nil, fmt.Errorf("invalid IPv4 address: %s", localIPNet)
}

func IPv4ToInt(ip net.IP) (uint32, error) {
	if ip4 := ip.To4(); ip4 != nil {
		return binary.BigEndian.Uint32(ip4), nil
	}
	return 0, fmt.Errorf("invalid IPv4 address: %s", ip)
}

func IPv4StrToInt(ip string) (uint32, error) {
	ipv4 := net.ParseIP(ip)
	if ipv4 == nil {
		return 0, fmt.Errorf("invalid IPv4 address: %s", ip)
	}
	return IPv4ToInt(ipv4)
}
