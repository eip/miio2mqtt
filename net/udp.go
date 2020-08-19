package net

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/eip/miio2mqtt/config"
	log "github.com/go-pkgz/lgr"
)

const udpNetwork = "udp4"

type UDPListener struct {
	Connection *net.UDPConn
	Packets    chan UDPPacket
}

type UDPPacket struct {
	Address net.UDPAddr
	Data    []byte
	Time    time.Time
}

func StartListener(ctx context.Context, wg *sync.WaitGroup) (*UDPListener, error) {
	laddr, err := GetLocalUDPAddr(0)
	if err != nil {
		return nil, err
	}
	listener := &UDPListener{}
	listener.Connection, err = net.ListenUDP(udpNetwork, laddr)
	if err != nil {
		return nil, err
	}
	chanLength := 4 * (len(config.C.Devices) + 1)
	listener.Packets = make(chan UDPPacket, chanLength)
	wg.Add(1)
	go func() {
		listenUDPPackets(ctx, listener)
		wg.Done()
	}()
	return listener, nil
}

func (l *UDPListener) Stop() {
	if err := l.Connection.Close(); err != nil {
		log.Printf("[ERROR] %v", err)
	}
	// log.Print("[DEBUG] UDP connection closed")
}

func (l *UDPListener) Purge() {
	count := 0
loop:
	for {
		select {
		case <-l.Packets:
			count++
		default:
			break loop
		}
	}
	if count > 0 {
		log.Printf("[DEBUG] %d packets purged", count)
	}
}

func listenUDPPackets(ctx context.Context, l *UDPListener) {
	log.Printf("[DEBUG] listening %v for UDP packets...", l.Connection.LocalAddr())
	// defer close(udpPackets)
	buffer := make([]byte, 1024)
	for {
		n, addr, err := l.Connection.ReadFromUDP(buffer)
		if err != nil {
			if nerr, ok := err.(*net.OpError); ok && nerr.Err.Error() == "use of closed network connection" {
				log.Print("[DEBUG] network connection closed")
				return
			}
			log.Printf("[WARN] %v", err)
			if ctx.Err() != nil {
				return
			}
			continue
		}
		pktTime := time.Now()
		select {
		case <-ctx.Done():
			log.Print("[DEBUG] stop listening for UDP packets")
			l.Stop()
			return
		case l.Packets <- UDPPacket{Address: *addr, Data: buffer[:n], Time: pktTime}:
			log.Printf("[DEBUG] %d bytes received from %v at %s", n, addr, pktTime.Format("15:04:05.999"))
		}
	}
}
