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

type UDPCommunicator struct {
	LocalAddress     *net.UDPAddr
	BroadcastAddress *net.UDPAddr
	Connection       *net.UDPConn
	Packets          chan UDPPacket
	ctx              context.Context
	cancel           context.CancelFunc
	config           *config.Config
}

type UDPPacket struct {
	Address   net.UDPAddr
	Data      []byte
	TimeStamp miio.TimeStamp
}

func NewCommunicator(config *config.Config) *UDPCommunicator {
	return &UDPCommunicator{config: config}
}

func (c *UDPCommunicator) Start(ctx context.Context, wg *sync.WaitGroup) error {
	var err error
	c.LocalAddress, c.BroadcastAddress, err = GetUDPAddresses(c.config.MiioPort)
	if err != nil {
		return err
	}
	c.Connection, err = net.ListenUDP(udpNetwork, c.LocalAddress)
	if err != nil {
		return err
	}
	chanLength := 4 * (len(c.config.Devices) + 1)
	c.Packets = make(chan UDPPacket, chanLength) // TODO check chan max length
	c.ctx, c.cancel = context.WithCancel(ctx)
	// defer listener.cancel()
	wg.Add(1)
	go func() { defer wg.Done(); c.listenUDPPackets() }()
	return nil
}

func (c *UDPCommunicator) Stop() {
	c.cancel()
	if err := c.Connection.Close(); err != nil {
		log.Printf("[ERROR] %v", err)
	}
	c.purge()
	c.Packets = nil
}

func (c *UDPCommunicator) purge() {
	count := 0
loop:
	for {
		select {
		case <-c.Packets:
			count++
		default:
			break loop
		}
	}
	if count > 0 {
		log.Printf("[DEBUG] %d packets purged", count)
	}
}

func (c *UDPCommunicator) listenUDPPackets() {
	log.Printf("[DEBUG] listening %v for UDP packets...", c.Connection.LocalAddr())
	// defer close(udpPackets)
	buffer := make([]byte, 1024)
	for {
		n, addr, err := c.Connection.ReadFromUDP(buffer)
		if err != nil {
			if nerr, ok := err.(*net.OpError); ok && nerr.Err.Error() == errNetClosingString {
				log.Print("[DEBUG] stop listening for UDP packets")
				return
			}
			log.Printf("[WARN] %v", err)
			if c.ctx.Err() != nil { // ctx was done
				log.Print("[DEBUG] stop listening for UDP packets")
				return
			}
			continue
		}
		pktTime := miio.Now()
		select {
		case <-c.ctx.Done():
			log.Print("[DEBUG] stop listening for UDP packets")
			c.Stop()
			return
		case c.Packets <- UDPPacket{Address: *addr, Data: buffer[:n], TimeStamp: pktTime}:
			log.Printf("[DEBUG] %d bytes received from %v", n, addr)
		}
	}
}
