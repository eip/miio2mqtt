package net

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/eip/miio2mqtt/config"
	h "github.com/eip/miio2mqtt/helpers"
	"github.com/eip/miio2mqtt/miio"
	log "github.com/go-pkgz/lgr"
)

type Poller struct {
	config    *config.Config
	transport *UDPTransport
	devices   miio.Devices
	updates   chan *miio.Device
}

func NewPoller(config *config.Config, transport *UDPTransport, devices miio.Devices) *Poller {
	updates := make(chan *miio.Device, 2*len(devices)) // TODO check chan max length
	return &Poller{config: config, transport: transport, devices: devices, updates: updates}
}

// PollDevices queries devices and updates devices info
func (p *Poller) PollDevices(ctx context.Context) error {
	left := p.devices.Count(miio.DeviceNeedsUpdate)
	if left == 0 {
		log.Print("[INFO] no device to update")
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, p.config.PollTimeout)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() { defer wg.Done(); p.sendPackets(ctx) }()

	log.Print("[DEBUG] start updating devices")
	var pkt UDPPacket
loop:
	for {
		select {
		case <-ctx.Done():
			log.Print("[DEBUG] updating devices done")
			break loop
		case pkt = <-p.transport.Packets:
		}
		if len(pkt.Data) == 32 {
			p.processHelloReply(pkt)
			continue
		}
		if ok := p.processReply(pkt); !ok {
			continue
		}
		left--
		if left == 0 {
			break loop
		}
	}
	cancel()
	err := error(nil)
	if left > 0 {
		err = ctx.Err()
	}
	wg.Wait()
	return err
}

func (p *Poller) Updates() <-chan *miio.Device {
	return p.updates
}

func (p *Poller) sendPackets(ctx context.Context) {
	helloPacket, _ := miio.NewHelloPacket().Encode(nil)
	next := time.Duration(p.config.PollTimeout / 50)
	log.Print("[DEBUG] start sending requests")
	for {
		select {
		case <-time.After(next):
			helloPacketSent := false
			anyPacketSent := false
			for _, d := range p.devices {
				if d.InFinalStage() {
					continue
				}
				switch d.Stage() {
				case miio.Undiscovered:
					if helloPacketSent {
						break
					}
					log.Printf("[DEBUG] sending hello packet to %v", p.transport.BroadcastAddress)
					if _, err := p.transport.Connection.WriteToUDP(helloPacket, p.transport.BroadcastAddress); err != nil {
						log.Printf("[WARN] %v", err)
						break
					}
					helloPacketSent = true
					anyPacketSent = true
				case miio.Found:
					addr := ParseUDPAddr(d.Address, p.config.MiioPort)
					if addr == nil {
						log.Printf("[WARN] invalid %s address: %s", d.Name, d.Address)
						break
					}
					info := p.config.Models.MiioInfo("*")
					if len(info) == 0 {
						break
					}
					req, data, err := d.Request([]byte(info))
					if err != nil {
						log.Printf("[WARN] %v", err)
						break
					}
					log.Printf("[DEBUG] sending %s to %s (%s)", req.Data, d.Name, addr)
					if _, err := p.transport.Connection.WriteToUDP(data, addr); err != nil {
						log.Printf("[WARN] %v", err)
						break
					}
					anyPacketSent = true
				case miio.Valid:
					addr := ParseUDPAddr(d.Address, p.config.MiioPort)
					if addr == nil {
						log.Printf("[WARN] invalid %s address: %s", d.Name, d.Address)
						break
					}
					getProp := p.config.Models.GetProp(d.Model())
					if len(getProp) == 0 {
						break
					}
					req, data, err := d.Request([]byte(getProp))
					if err != nil {
						log.Printf("[WARN] %v", err)
						break
					}
					log.Printf("[DEBUG] sending %s to %s (%s)", req.Data, d.Name, addr)
					if _, err := p.transport.Connection.WriteToUDP(data, addr); err != nil {
						log.Printf("[WARN] %v", err)
						break
					}
					anyPacketSent = true
				}
			}
			if !anyPacketSent { // all devices were updated
				log.Print("[DEBUG] no devices to send requests left")
				return
			}
			next = time.Duration(p.config.PollTimeout / 5)
		case <-ctx.Done():
			log.Print("[DEBUG] stop sending requests")
			return
		}
	}
}

func (p *Poller) processHelloReply(pkt UDPPacket) bool {
	did, iaddr, saddr, err := getDeviceIDAndAddress(pkt)
	if err != nil {
		log.Printf("[WARN] invalid packet received from %s: %x (%v)", saddr, pkt.Data, err)
		return false
	}
	reply, err := miio.Decode(pkt.Data, nil)
	if err != nil {
		log.Printf("[WARN] invalid packet received from %s: %x (%v)", saddr, pkt.Data, err)
		return false
	}
	updateDID := false
	d, ok := p.devices[did]
	if !ok {
		updateDID = true
		d, ok = p.devices[iaddr]
	}
	if !ok {
		log.Printf("[DEBUG] hello reply from unknown device %08x (%s)", did, saddr)
		return false
	}
	if miio.DeviceFound(d) {
		log.Printf("[DEBUG] hello reply from already discovered %s", d.Name)
		return false
	}
	log.Printf("[DEBUG] hello reply from %s (stage=%s): %v", d.Name, d.Stage(), reply)
	d.SetTimeShift(pkt.TimeStamp, reply.TimeStamp)
	if updateDID {
		d.ID = did
		p.devices[did] = d
		delete(p.devices, iaddr)
	} else {
		d.Address = saddr
	}
	d.SetStage(miio.Found)
	log.Printf("[INFO] discovered %s: %08x (%s)", d.Name, d.ID, d.Address)
	return true
}

func (p *Poller) processReply(pkt UDPPacket) bool {
	did, _, saddr, err := getDeviceIDAndAddress(pkt)
	if err != nil {
		log.Printf("[WARN] invalid packet received from %s: %x (%v)", saddr, pkt.Data, err)
		return false
	}
	d, ok := p.devices[did]
	if !ok {
		log.Printf("[DEBUG] reply from unknown device %08x (%s)", did, saddr)
		return false
	}
	if d.InFinalStage() {
		log.Printf("[DEBUG] reply from already updated %s", d.Name)
		return false
	}
	reply, err := miio.Decode(pkt.Data, d.Token())
	if err != nil {
		log.Printf("[WARN] unable to decode packet from %s: %x (%v)", d.Name, pkt.Data, err)
		return false
	}
	log.Printf("[DEBUG] reply from %s (stage=%s): %s", d.Name, d.Stage(), reply.Data)

	parsed := miio.ParseReply(reply.Data)
	switch parsed.Type {
	case miio.MiioInfo:
		if d.InStage(miio.Valid) {
			log.Printf("[DEBUG] reply from already identified %s: %s", d.Name, reply.Data)
			return false
		}
		d.SetModel(parsed.Model)
		d.SetStage(miio.Valid)
		log.Printf("[INFO] identified %s model: %s", d.Name, d.Model())
		return d.InFinalStage()
	case miio.GetProp:
		if d.InFinalStage() {
			log.Printf("[DEBUG] reply from already updated %s: %s", d.Name, reply.Data)
			return false
		}
		newProps, err := p.buildDeviceProperties(d, parsed.Props)
		if err != nil {
			log.Printf("[WARN] %v", err)
			return false
		}
		oldProps := d.Properties()
		stateChanged := newProps != oldProps
		if stateChanged {
			d.SetProperties(newProps)
			d.SetStateChangedNow()
			if len(oldProps) > 0 {
				newProps = h.DiffStrings(h.StripJSONQuotes(oldProps), h.StripJSONQuotes(newProps), "96")
			} else {
				newProps = h.StripJSONQuotes(newProps)
			}
			log.Printf("[INFO] updated %s: %s", d.Name, newProps)
		} else {
			log.Printf("[INFO] %s state unchanged", d.Name)
		}
		d.SetTimeShift(pkt.TimeStamp, reply.TimeStamp)
		d.SetUpdatedNow()
		d.SetStage(miio.Updated)
		if d.StateChangeUnpublished() {
			p.updates <- d
		}
		return true
	default:
		log.Printf("[WARN] unable to parse device reply: %v", reply)
	}
	return false
}

func (p *Poller) buildDeviceProperties(d *miio.Device, props []interface{}) (string, error) {
	params := p.config.Models.Params(d.Model())
	if len(props) != len(params) {
		return "", fmt.Errorf("invalid number of properties (%d of %d) for %s (%s)", len(props), len(params), d.Name, d.Model())
	}
	data := map[string]interface{}{}
	for i, key := range params {
		data[key] = p.fixProperty(props[i])
	}
	result, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("unable to encode properties: %#v", data)
	}
	return string(result), nil
}

func (p *Poller) fixProperty(value interface{}) interface{} {
	if fixed, ok := p.config.Properties[value]; ok {
		return fixed
	}
	return value
}

func getDeviceIDAndAddress(pkt UDPPacket) (did uint32, iaddr uint32, saddr string, err error) {
	saddr = pkt.Address.IP.String()
	did, err = miio.GetDeviceID(pkt.Data)
	if err != nil {
		return
	}
	iaddr, err = IPv4ToInt(pkt.Address.IP)
	return
}
