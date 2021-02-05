package net

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/eip/miio2mqtt/config"
	"github.com/eip/miio2mqtt/miio"
	"github.com/eip/miio2mqtt/mqtt"
	log "github.com/go-pkgz/lgr"
)

// PollDevices queries devices and updates devices info
func PollDevices(ctx context.Context, listener *UDPListener, devices miio.Devices, messages chan<- mqtt.Message) error {
	left := devices.Count(miio.DeviceNeedsUpdate)
	if left == 0 {
		log.Print("[INFO] no device to update")
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, config.C.PollTimeout)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() { defer wg.Done(); sendPackets(ctx, listener, devices) }()

	log.Print("[DEBUG] start updating devices")
	var pkt UDPPacket
loop:
	for {
		select {
		case <-ctx.Done():
			log.Print("[DEBUG] updating devices done")
			break loop
		case pkt = <-listener.Packets:
		}
		if len(pkt.Data) == 32 {
			processHelloReply(pkt, devices)
			continue
		}
		if ok := processReply(pkt, devices, messages); !ok {
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

func sendPackets(ctx context.Context, listener *UDPListener, devices miio.Devices) {
	helloPacket, _ := miio.NewHelloPacket().Encode(nil)
	next := time.Duration(config.C.PollTimeout / 50)
	log.Print("[DEBUG] start sending requests")
	for {
		select {
		case <-time.After(next):
			helloPacketSent := false
			anyPacketSent := false
			for _, d := range devices {
				switch d.GetStage() {
				case miio.Undiscovered:
					if helloPacketSent {
						break
					}
					log.Printf("[DEBUG] sending hello packet to %v", listener.BroadcastAddress)
					if _, err := listener.Connection.WriteToUDP(helloPacket, listener.BroadcastAddress); err != nil {
						log.Printf("[WARN] %v", err)
						break
					}
					helloPacketSent = true
					anyPacketSent = true
				case miio.Found:
					addr := ParseUDPAddr(d.Address, config.C.MiioPort)
					if addr == nil {
						log.Printf("[WARN] invalid %s address: %s", d.Name, d.Address)
						break
					}
					info := config.C.Models.MiioInfo("*")
					if len(info) == 0 {
						break
					}
					req, data, err := d.Request([]byte(info))
					if err != nil {
						log.Printf("[WARN] %v", err)
						break
					}
					log.Printf("[DEBUG] sending %s to %s (%s)", req.Data, d.Name, addr)
					if _, err := listener.Connection.WriteToUDP(data, addr); err != nil {
						log.Printf("[WARN] %v", err)
						break
					}
					anyPacketSent = true
				case miio.Valid:
					addr := ParseUDPAddr(d.Address, config.C.MiioPort)
					if addr == nil {
						log.Printf("[WARN] invalid %s address: %s", d.Name, d.Address)
						break
					}
					getProp := config.C.Models.GetProp(d.Model)
					if len(getProp) == 0 {
						break
					}
					req, data, err := d.Request([]byte(getProp))
					if err != nil {
						log.Printf("[WARN] %v", err)
						break
					}
					log.Printf("[DEBUG] sending %s to %s (%s)", req.Data, d.Name, addr)
					if _, err := listener.Connection.WriteToUDP(data, addr); err != nil {
						log.Printf("[WARN] %v", err)
						break
					}
					anyPacketSent = true
				case miio.Updated:
					break
				}
			}
			if !anyPacketSent { // all devices were updated
				log.Print("[DEBUG] no devices to send requests left")
				return
			}
			next = time.Duration(config.C.PollTimeout / 5)
		case <-ctx.Done():
			log.Print("[DEBUG] stop sending requests")
			return
		}
	}
}

func processHelloReply(pkt UDPPacket, devices miio.Devices) bool {
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
	d, ok := devices[did]
	if !ok {
		updateDID = true
		d, ok = devices[iaddr]
	}
	if !ok {
		log.Printf("[DEBUG] hello reply from unknown device %08x (%s)", did, saddr)
		return false
	}
	if miio.DeviceFound(d) {
		log.Printf("[DEBUG] hello reply from already discovered %s", d.Name)
		return false
	}
	log.Printf("[DEBUG] hello reply from %s (stage=%s): %v", d.Name, d.GetStage(), reply)
	d.SetTimeShift(pkt.TimeStamp, reply.TimeStamp)
	if updateDID {
		d.ID = did
		devices[did] = d
		delete(devices, iaddr)
	} else {
		d.Address = saddr
	}
	d.SetStage(miio.Found)
	log.Printf("[INFO] discovered %s: %08x (%s)", d.Name, d.ID, d.Address)
	return true
}

func processReply(pkt UDPPacket, devices miio.Devices, messages chan<- mqtt.Message) bool {
	did, _, saddr, err := getDeviceIDAndAddress(pkt)
	if err != nil {
		log.Printf("[WARN] invalid packet received from %s: %x (%v)", saddr, pkt.Data, err)
		return false
	}
	d, ok := devices[did]
	if !ok {
		log.Printf("[DEBUG] reply from unknown device %08x (%s)", did, saddr)
		return false
	}
	if miio.DeviceUpdated(d) {
		log.Printf("[DEBUG] reply from already updated %s", d.Name)
		return false
	}
	reply, err := miio.Decode(pkt.Data, d.Token[:])
	if err != nil {
		log.Printf("[WARN] unable to decode packet from %s: %x (%v)", d.Name, pkt.Data, err)
		return false
	}
	log.Printf("[DEBUG] reply from %s (stage=%s): %s", d.Name, d.GetStage(), reply.Data)

	// replyTS := int64(reply.Stamp) * 1e9
	// deviceTS, _ := d.GetTimeStamp(pkt.Time)
	// deviceTS = deviceTS / 1e6 * 1e6
	// if h.TimeStampDiff(replyTS, deviceTS) > time.Millisecond {
	// 	log.Printf("[INFO] last reply time %v not in sync with device time %v [%v]", time.Duration(replyTS), time.Duration(deviceTS), time.Duration(replyTS-deviceTS))
	// }

	parsed := miio.ParseReply(reply.Data)
	switch parsed.Type {
	case miio.MiioInfo:
		if miio.DeviceValid(d) {
			log.Printf("[DEBUG] reply from already identified %s: %s", d.Name, reply.Data)
			return false
		}
		d.Model = parsed.Model
		d.SetStage(miio.Valid)
		log.Printf("[INFO] identified %s model: %s", d.Name, d.Model)
		return false
	case miio.GetProp:
		if miio.DeviceUpdated(d) {
			log.Printf("[DEBUG] reply from already updated %s: %s", d.Name, reply.Data)
			return false
		}
		params := config.C.Models.Params(d.Model)
		if len(parsed.Props) != len(params) {
			log.Printf("[WARN] invalid number of propertiies (%d of %d) for %s (%s)", len(parsed.Props), len(params), d.Name, d.Model)
			return false
		}
		data := map[string]interface{}{}
		for i, key := range params {
			data[key] = fixProperty(parsed.Props[i])
		}
		props, err := json.Marshal(data)
		if err != nil {
			log.Printf("[WARN] unable to encode properties: %#v", data)
			return false
		}
		stateChanged := string(props) != d.Properties
		if stateChanged {
			d.Properties = string(props)
			d.SetStateChangedNow()
			log.Printf("[INFO] updated %s: %s", d.Name, d.Properties)
		} else {
			log.Printf("[INFO] %s state unchanged", d.Name)
		}
		d.SetTimeShift(pkt.TimeStamp, reply.TimeStamp)
		d.SetUpdatedNow()
		d.SetStage(miio.Updated)
		if d.StateChangeUnpublished() {
			messages <- mqtt.Message{Device: d}
		}
		return true
	default:
		log.Printf("[WARN] unable to parse device reply: %v", reply)
	}
	return false
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

func fixProperty(value interface{}) interface{} {
	if fixed, ok := config.C.Properties[value]; ok {
		return fixed
	}
	return value
}
