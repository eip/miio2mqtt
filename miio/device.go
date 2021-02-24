package miio

import (
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sync"

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
	sync.Mutex
	DeviceCfg
	Name             string
	model            string
	token            [16]byte
	properties       string
	timeShift        TimeStamp
	stage            DeviceStage
	finalStage       DeviceStage
	requestID        uint32
	updatedAt        TimeStamp
	stateChangedAt   TimeStamp
	statePublishedAt TimeStamp
}

type Devices map[uint32]*Device

type CheckDevice func(d *Device) bool

func NewDevice(cfg DeviceCfg, name string) *Device {
	d := Device{
		DeviceCfg: cfg,
		Name:      name,
	}
	token, _ := hex.DecodeString(cfg.Token)
	copy(d.token[:], token)
	d.SetStage(Undiscovered)
	d.SetFinalStage(Updated)
	return &d
}

func (d *Device) Request(data []byte) (*Packet, []byte, error) {
	timeStamp, err := d.Now()
	if err != nil {
		return nil, nil, err
	}
	d.Lock()
	defer d.Unlock()
	d.requestID++
	return deviceRequest(data, d.ID, d.requestID, timeStamp, d.token[:])
}

func deviceRequest(data []byte, deviceID uint32, requestID uint32, timeStamp TimeStamp, token []byte) (*Packet, []byte, error) {
	pkt := NewPacket(deviceID, timeStamp, reID.ReplaceAll(data, []byte(fmt.Sprintf("${1}%d", requestID))))
	raw, err := pkt.Encode(token)
	if err != nil {
		return pkt, nil, err
	}
	return pkt, raw, nil
}

func (d *Device) Model() string {
	d.Lock()
	defer d.Unlock()
	return d.model
}

func (d *Device) SetModel(model string) {
	d.Lock()
	d.model = model
	d.Unlock()
}

func (d *Device) Token() []byte {
	d.Lock()
	defer d.Unlock()
	return d.token[:]
}

func (d *Device) Properties() string {
	d.Lock()
	defer d.Unlock()
	return d.properties
}

func (d *Device) SetProperties(properties string) {
	d.Lock()
	d.properties = properties
	d.Unlock()
}

func (d *Device) Stage() DeviceStage {
	d.Lock()
	defer d.Unlock()
	return d.stage
}

func (d *Device) SetStage(stage DeviceStage) {
	if stage < Undiscovered || stage > Updated {
		stage = Undiscovered
	}
	d.Lock()
	d.stage = stage
	d.Unlock()
}

func (d *Device) FinalStage() DeviceStage {
	d.Lock()
	defer d.Unlock()
	return d.finalStage
}

func (d *Device) SetFinalStage(stage DeviceStage) {
	if stage < Undiscovered || stage > Updated {
		stage = Undiscovered
	}
	d.Lock()
	d.finalStage = stage
	d.Unlock()
}

func (d *Device) InStage(stage DeviceStage) bool {
	d.Lock()
	defer d.Unlock()
	return d.stage >= stage
}

func (d *Device) InFinalStage() bool {
	d.Lock()
	defer d.Unlock()
	return d.stage >= d.finalStage
}

func (d *Device) TimeStamp(now TimeStamp) (TimeStamp, error) {
	d.Lock()
	ts := d.timeShift
	d.Unlock()
	if ts == 0 {
		return 0, errors.New("device time shift is not set")
	}
	if ts >= now {
		return 0, errors.New("invalid device time shift")
	}
	return now - ts, nil
}

func (d *Device) Now() (TimeStamp, error) {
	return d.TimeStamp(Now())
}

func (d *Device) SetTimeShift(now TimeStamp, replyTS TimeStamp) error {
	if replyTS >= now {
		return errors.New("device time cannot be in future")
	}
	ts := now - replyTS
	d.Lock()
	d.timeShift = ts
	d.Unlock()
	return nil
}

func (d *Device) UpdatedAt() TimeStamp {
	d.Lock()
	defer d.Unlock()
	return d.updatedAt
}

func (d *Device) SetUpdatedNow() {
	now := Now()
	d.Lock()
	d.updatedAt = now
	d.Unlock()
}

func (d *Device) UpdatedIn() TimeStamp {
	d.Lock()
	ts := d.updatedAt
	d.Unlock()
	now := Now()
	if ts == 0 || now <= TimeStamp(ts) {
		return 0
	}
	return now - TimeStamp(ts)
}

func (d *Device) SetStateChangedNow() {
	now := Now()
	d.Lock()
	d.stateChangedAt = now
	d.Unlock()
}

func (d *Device) SetStatePublishedNow() {
	now := Now()
	d.Lock()
	d.statePublishedAt = now
	d.Unlock()
}

func (d *Device) StateChangeUnpublished() bool {
	d.Lock()
	cts := d.stateChangedAt
	pts := d.statePublishedAt
	d.Unlock()
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
	return d.Stage() >= Found
}

func DeviceValid(d *Device) bool {
	return d.Stage() >= Valid
}

func DeviceUpdated(d *Device) bool {
	return d.Stage() >= Updated
}

func AnyDevice(_ *Device) bool {
	return true
}

func DeviceNeedsUpdate(d *Device) bool {
	return d.Stage() < Updated
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
