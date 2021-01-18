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
const errNetClosingString = "use of closed network connection" // defined in internal/poll package

type UDPListener struct {
	Connection *net.UDPConn
	Packets    chan UDPPacket
	ctx        context.Context
	cancel     context.CancelFunc
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
	listener.ctx, listener.cancel = context.WithCancel(ctx)
	// defer listener.cancel()
	wg.Add(1)
	go func() { defer wg.Done(); listenUDPPackets(listener) }()
	return listener, nil
}

func (l *UDPListener) Stop() {
	l.cancel()
	if err := l.Connection.Close(); err != nil {
		log.Printf("[ERROR] %v", err)
	}
	l.purge()
	l.Packets = nil
}

func (l *UDPListener) purge() {
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

func listenUDPPackets(l *UDPListener) {
	log.Printf("[DEBUG] listening %v for UDP packets...", l.Connection.LocalAddr())
	// defer close(udpPackets)
	buffer := make([]byte, 1024)
	for {
		n, addr, err := l.Connection.ReadFromUDP(buffer)
		if err != nil {
			if nerr, ok := err.(*net.OpError); ok && nerr.Err.Error() == errNetClosingString {
				log.Print("[DEBUG] stop listening for UDP packets")
				return
			}
			log.Printf("[WARN] %v", err)
			if l.ctx.Err() != nil { // ctx was done
				log.Print("[DEBUG] stop listening for UDP packets")
				return
			}
			continue
		}
		pktTime := time.Now()
		select {
		case <-l.ctx.Done():
			log.Print("[DEBUG] stop listening for UDP packets")
			l.Stop()
			return
		case l.Packets <- UDPPacket{Address: *addr, Data: buffer[:n], Time: pktTime}:
			log.Printf("[DEBUG] %d bytes received from %v at %s", n, addr, pktTime.Format("15:04:05.999"))
		}
	}
}
