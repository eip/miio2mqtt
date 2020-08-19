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

func GetLocalUDPAddr(port int) (*net.UDPAddr, error) {
	conn, err := net.Dial(udpNetwork, probeAddress)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	addr := conn.LocalAddr().(*net.UDPAddr)
	addr.Port = port
	return addr, nil
}

func GetBroadcastUDPAddr(localAddr *net.UDPAddr, port int) (*net.UDPAddr, error) {
	if localAddr == nil || localAddr.IP == nil {
		var err error
		localAddr, err = GetLocalUDPAddr(port)
		if err != nil {
			return nil, err
		}
	}
	addr := &net.UDPAddr{IP: make(net.IP, net.IPv4len), Port: port, Zone: localAddr.Zone}
	copy(addr.IP, localAddr.IP)
	addr.IP[3] = 0xff
	return addr, nil
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
