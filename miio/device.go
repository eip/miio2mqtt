package miio

import (
	"errors"
	"fmt"
	"regexp"
	"sync/atomic"

	log "github.com/go-pkgz/lgr"
)

var reID = regexp.MustCompile(`("id":\s?)#+`)

// DeviceCfg represents a miIO configurable device properties
type DeviceCfg struct {
	Address string `yaml:"Address"`
	ID      uint32 `yaml:"ID"`
	Topic   string `yaml:"Topic"`
	Token   string `yaml:"Token"`
}

type DeviceStage int32

func (d DeviceStage) String() string {
	switch d {
	case 0:
		return "undiscovered"
	case 1:
		return "found"
	case 2:
		return "valid"
	case 3:
		return "updated"
	default:
		return "INVALID"
	}
}

const (
	Undiscovered DeviceStage = iota
	Found
	Valid
	Updated
)

// Device represents a miIO all device properties
type Device struct {
	DeviceCfg
	Name             string
	Model            string
	Token            [16]byte
	Properties       string
	timeShift        uint32
	stage            DeviceStage
	requestID        uint32
	updatedAt        TimeStamp
	stateChangedAt   TimeStamp
	statePublishedAt TimeStamp
}

type Devices map[uint32]*Device

type CheckDevice func(d *Device) bool

func (d *Device) Request(data []byte) (*Packet, []byte, error) {
	timeStamp, err := d.Now()
	if err != nil {
		return nil, nil, err
	}
	return deviceRequest(data, d.ID, atomic.AddUint32(&d.requestID, 1), timeStamp, d.Token[:])
}

func deviceRequest(data []byte, deviceID uint32, requestID uint32, timeStamp TimeStamp, token []byte) (*Packet, []byte, error) {
	pkt := NewPacket(deviceID, timeStamp, reID.ReplaceAll(data, []byte(fmt.Sprintf("${1}%d", requestID))))
	raw, err := pkt.Encode(token)
	if err != nil {
		return pkt, nil, err
	}
	return pkt, raw, nil
}

func (d *Device) GetStage() DeviceStage {
	result := atomic.LoadInt32((*int32)(&d.stage))
	return DeviceStage(result)
}

func (d *Device) SetStage(stage DeviceStage) {
	if stage < Undiscovered || stage > Updated {
		stage = Undiscovered
	}
	atomic.StoreInt32((*int32)(&d.stage), int32(stage))
}

func (d *Device) GetTimeStamp(now TimeStamp) (TimeStamp, error) {
	ts := atomic.LoadUint32(&d.timeShift)
	if ts == 0 {
		return 0, errors.New("device time shift is not set")
	}
	return now - TimeStamp(ts), nil
}

func (d *Device) Now() (TimeStamp, error) {
	return d.GetTimeStamp(Now())
}

func (d *Device) SetTimeShift(now TimeStamp, replyTS TimeStamp) error {
	if replyTS >= now {
		return errors.New("device time cannot be in future")
	}
	ts := now - replyTS
	atomic.StoreUint32(&d.timeShift, uint32(ts))
	return nil
}

func (d *Device) UpdatedAt() TimeStamp {
	ts := atomic.LoadUint32((*uint32)(&d.updatedAt))
	return TimeStamp(ts)
}

func (d *Device) SetUpdatedNow() {
	now := Now()
	atomic.StoreUint32((*uint32)(&d.updatedAt), uint32(now))
}

func (d *Device) UpdatedIn() TimeStamp {
	ts := atomic.LoadUint32((*uint32)(&d.updatedAt))
	now := Now()
	if ts == 0 || now <= TimeStamp(ts) {
		return 0
	}
	return now - TimeStamp(ts)
}

func (d *Device) SetStateChangedNow() {
	now := Now()
	atomic.StoreUint32((*uint32)(&d.stateChangedAt), uint32(now))
}

func (d *Device) SetStatePublishedNow() {
	now := Now()
	atomic.StoreUint32((*uint32)(&d.statePublishedAt), uint32(now))
}

func (d *Device) StateChangeUnpublished() bool {
	cts := atomic.LoadUint32((*uint32)(&d.stateChangedAt))
	pts := atomic.LoadUint32((*uint32)(&d.statePublishedAt))
	return cts > pts
}

func (dm Devices) Count(valid CheckDevice) int {
	result := 0
	for _, d := range dm {
		if valid(d) {
			result++
		}
	}
	return result
}

func (dm Devices) SetStage(stage DeviceStage, check CheckDevice) {
	for _, d := range dm {
		if check(d) {
			d.SetStage(stage)
		}
	}
}

func DeviceFound(d *Device) bool {
	return d.GetStage() >= Found
}

func DeviceValid(d *Device) bool {
	return d.GetStage() >= Valid
}

func DeviceUpdated(d *Device) bool {
	return d.GetStage() >= Updated
}

func AnyDevice(_ *Device) bool {
	return true
}

func DeviceNeedsUpdate(d *Device) bool {
	return d.GetStage() < Updated
}

func DeviceOutdated(timeout TimeStamp) CheckDevice {
	return func(d *Device) bool {
		if !DeviceFound(d) {
			return false
		}
		updatedIn := d.UpdatedIn()
		if updatedIn > timeout {
			log.Printf("[INFO] outdated %s (updated %v ago)", d.Name, updatedIn)
			return true
		}
		return false
	}
}
