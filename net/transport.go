package net

import (
	"context"
	"net"
	"sync"

	"github.com/eip/miio2mqtt/config"
	"github.com/eip/miio2mqtt/miio"
	log "github.com/go-pkgz/lgr"
)

const udpNetwork = "udp4"
const errNetClosingString = "use of closed network connection" // defined in internal/poll package

type UDPTransport struct {
	LocalAddress     *net.UDPAddr
	BroadcastAddress *net.UDPAddr
	Connection       *net.UDPConn
	config           *config.Config
	packets          chan *UDPPacket
	cancel           context.CancelFunc
}

type UDPPacket struct {
	Address   net.UDPAddr
	Data      []byte
	TimeStamp miio.TimeStamp
}

func NewTransport(config *config.Config) *UDPTransport {
	return &UDPTransport{config: config}
}

func (t *UDPTransport) Start(ctx context.Context, wg *sync.WaitGroup) error {
	var err error
	t.LocalAddress, t.BroadcastAddress, err = GetUDPAddresses(t.config.MiioPort)
	if err != nil {
		return err
	}
	t.Connection, err = net.ListenUDP(udpNetwork, t.LocalAddress)
	if err != nil {
		return err
	}
	t.packets = make(chan *UDPPacket, 1+2*len(t.config.Devices)) // TODO check chan max length
	ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	// defer listener.cancel()
	wg.Add(1)
	go func() { defer wg.Done(); t.listenUDPPackets(ctx) }()
	return nil
}

func (t *UDPTransport) Stop() {
	t.cancel()
	if err := t.Connection.Close(); err != nil {
		log.Printf("[ERROR] %v", err)
	}
	t.purgePackets()
	t.packets = nil
}

func (t *UDPTransport) Packets() <-chan *UDPPacket {
	return t.packets
}

func (t *UDPTransport) purgePackets() {
	count := 0
loop:
	for {
		select {
		case <-t.packets:
			count++
		default:
			break loop
		}
	}
	if count > 0 {
		log.Printf("[DEBUG] %d packets purged", count)
	}
}

func (t *UDPTransport) listenUDPPackets(ctx context.Context) {
	log.Printf("[DEBUG] listening %v for UDP packets...", t.Connection.LocalAddr())
	// defer close(udpPackets)
	buffer := make([]byte, 1024)
	for {
		n, addr, err := t.Connection.ReadFromUDP(buffer)
		if err != nil {
			if nerr, ok := err.(*net.OpError); ok && nerr.Err.Error() == errNetClosingString {
				log.Print("[DEBUG] stop listening for UDP packets")
				return
			}
			log.Printf("[WARN] %v", err)
			if ctx.Err() != nil { // ctx was done
				log.Print("[DEBUG] stop listening for UDP packets")
				return
			}
			continue
		}
		pktTime := miio.Now()
		select {
		case <-ctx.Done():
			log.Print("[DEBUG] stop listening for UDP packets")
			t.Stop()
			return
		case t.packets <- &UDPPacket{Address: *addr, Data: buffer[:n], TimeStamp: pktTime}:
			log.Printf("[DEBUG] %d bytes received from %v", n, addr)
		}
	}
}
