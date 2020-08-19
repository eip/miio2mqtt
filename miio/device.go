package miio

import (
	"errors"
	"fmt"
	"regexp"
	"sync/atomic"
	"time"
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
	Name       string
	Model      string
	Token      [16]byte
	Properties string
	UpdatedAt  time.Time
	PushedAt   time.Time
	TimeShift  time.Duration
	RequestID  uint32
	stage      int32
	// complete  chan struct{}
}

type Devices map[uint32]*Device

type CheckDevice func(d *Device) bool

// func (d *Device) IsValid() bool {
// 	return d.TimeShift > 0
// }

func (d *Device) Now() (time.Time, error) {
	if d.TimeShift == 0 {
		return time.Unix(0, 0), errors.New("device time shift is not set")
	}
	return time.Now().Add(-d.TimeShift), nil
}

func (d *Device) Request(data []byte) (*Packet, []byte, error) {
	deviceTime, err := d.Now()
	if err != nil {
		return nil, nil, err
	}
	d.RequestID++
	pkt := NewPacket(d.ID, deviceTime, reID.ReplaceAll(data, []byte(fmt.Sprintf("${1}%d", d.RequestID))))
	// pkte := *pkt
	raw, err := pkt.Encode(d.Token[:])
	if err != nil {
		return pkt, nil, err
	}
	return pkt, raw, nil
}

func (d *Device) GetStage() DeviceStage {
	result := atomic.LoadInt32(&d.stage)
	return DeviceStage(result)
}

func (d *Device) SetStage(stage DeviceStage) {
	atomic.StoreInt32(&d.stage, int32(stage))
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

func DeviceNeedUpdate(d *Device) bool {
	return d.GetStage() < Updated
}
