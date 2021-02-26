package miio

import (
	"errors"
	"regexp"
	"testing"

	h "github.com/eip/miio2mqtt/helpers"
)

var testLog = h.InitTestLog()

func TestDeviceStage_String(t *testing.T) {
	tests := []struct {
		name string
		d    DeviceStage
		want string
	}{
		{name: "undiscovered", d: Undiscovered, want: "undiscovered"},
		{name: "found", d: Found, want: "found"},
		{name: "valid", d: Valid, want: "valid"},
		{name: "updated", d: Updated, want: "updated"},
		{name: "invalid 1", d: -1, want: "INVALID"},
		{name: "invalid 2", d: 4, want: "INVALID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.d.String()
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_NewDevice(t *testing.T) {
	cfg := DeviceCfg{
		Address: "192.168.0.1",
		ID:      0x00112233,
		Topic:   "home/devices/test",
		Token:   "00112233445566778899aabbccddeeff",
	}
	want := &Device{
		DeviceCfg:  cfg,
		Name:       "Test Device",
		stage:      0,
		finalStage: 3,
	}
	copy(want.token[:], h.FromHex("00112233445566778899aabbccddeeff"))
	got := NewDevice(cfg, "Test Device")
	h.AssertEqual(t, got, want)
}

func TestDevice_Request(t *testing.T) {
	tests := []struct {
		name          string
		device        *Device
		data          string
		wantRequestID uint32
		wantPkt       *Packet
		err           error
	}{
		{
			name:   "Device time shift is not set",
			device: &Device{},
			err:    errors.New("device time shift is not set"),
		},
		{
			name:          "Real Request",
			device:        &Device{DeviceCfg: DeviceCfg{ID: 0x00112233}, token: [16]byte{}, timeShift: 1000 * hour, requestID: 122},
			data:          `{"method":"miIO.info","params":[],"id":#}`,
			wantRequestID: 123,
			wantPkt:       NewPacket(0x00112233, Now()-1000*hour, []byte(`{"method":"miIO.info","params":[],"id":123}`)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPkt, _, err := tt.device.Request([]byte(tt.data))
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, tt.device.requestID, tt.wantRequestID)
			if gotPkt != nil && TimeStampDiff(gotPkt.TimeStamp, tt.wantPkt.TimeStamp) <= 1*sec {
				tt.wantPkt.TimeStamp = gotPkt.TimeStamp
			}
			h.AssertEqual(t, gotPkt, tt.wantPkt)
		})
	}
}

func Test_deviceRequest(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		deviceID  uint32
		requestID uint32
		timeStamp TimeStamp
		token     []byte
		wantPkt   *Packet
		wantData  []byte
		err       error
	}{
		{
			name:      "Sample Request",
			data:      "123456789@ABCDEFGHI", // cspell: disable-line
			deviceID:  0x00112233,
			requestID: 0,
			timeStamp: sampleTS,
			token:     h.FromHex("00112233445566778899aabbccddeeff"),
			wantPkt:   NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
			wantData:  h.FromHex("21310040000000000011223300061e39b0cbb8837ed9a65a70165f2b7b4102722b487e7eed802b7df35c224caab8d216e43262c38b9cc073782c148668387d9e"),
		},
		{
			name:      "Invalid Token",
			data:      "123456789@ABCDEFGHI", // cspell: disable-line
			deviceID:  0x00112233,
			requestID: 0,
			timeStamp: sampleTS,
			token:     h.FromHex("00112233445566778899aabbccddeeff00"),
			wantPkt:   NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
			err:       errInvalidTokenLength,
		},
		{
			name:      "Real Request",
			data:      `{"method":"miIO.info","params":[],"id":#}`,
			deviceID:  0x00112233,
			requestID: 123,
			timeStamp: sampleTS,
			token:     h.FromHex("00112233445566778899aabbccddeeff"),
			wantPkt:   NewPacket(0x00112233, sampleTS, []byte(`{"method":"miIO.info","params":[],"id":123}`)),
			wantData:  h.FromHex("21310050000000000011223300061e39bc379b48c96b52ffd80dcbd9153594d12f42719f20d1969cd734b11bee043ad5a740d19c6e38ff8438a641c565d7b6f68c0c7008b88bc6869531a7ceac7818e2"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPkt, gotData, err := deviceRequest([]byte(tt.data), tt.deviceID, tt.requestID, tt.timeStamp, tt.token)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, gotPkt, tt.wantPkt)
			// h.AssertEqual(t, gotPkt.Data, tt.wantPkt.Data)
			h.AssertEqual(t, gotData, tt.wantData)
		})
	}
}

func TestDevice_Model(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   string
	}{
		{name: "Empty", device: &Device{}, want: ""},
		{name: "Dummy", device: &Device{model: "dummy.test.v1"}, want: "dummy.test.v1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.Model()
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestDevice_SetModel(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		model  string
		want   string
	}{
		{name: "Empty", device: &Device{model: "dummy.test.v2"}, model: "", want: ""},
		{name: "Dummy", device: &Device{model: "dummy.test.v2"}, model: "dummy.test.v1", want: "dummy.test.v1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.device.SetModel(tt.model)
			h.AssertEqual(t, tt.device.model, tt.want)
		})
	}
}

func TestDevice_Token(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   []byte
	}{
		{name: "Empty", device: &Device{}, want: h.FromHex("00000000000000000000000000000000")},
		{name: "Dummy", device: func() *Device {
			d := &Device{}
			copy(d.token[:], h.FromHex("00112233445566778899aabbccddeeff"))
			return d
		}(), want: h.FromHex("00112233445566778899aabbccddeeff")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.Token()
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestDevice_Properties(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   string
	}{
		{name: "Empty", device: &Device{}, want: ""},
		{name: "Dummy", device: &Device{properties: `{"value":123,"battery":100,"power":1,"state":1}`}, want: `{"value":123,"battery":100,"power":1,"state":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.Properties()
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestDevice_SetProperties(t *testing.T) {
	tests := []struct {
		name       string
		device     *Device
		properties string
		want       string
	}{
		{name: "Empty", device: &Device{properties: `{"value":456,"battery":85,"power":1,"state":0}`}, properties: "", want: ""},
		{name: "Dummy", device: &Device{properties: `{"value":456,"battery":85,"power":1,"state":0}`}, properties: `{"value":123,"battery":100,"power":1,"state":1}`, want: `{"value":123,"battery":100,"power":1,"state":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.device.SetProperties(tt.properties)
			h.AssertEqual(t, tt.device.properties, tt.want)
		})
	}
}

func TestDevice_GetTimeStamp(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		now    TimeStamp
		want   TimeStamp
		err    error
	}{
		{
			name:   "Device time shift is not set",
			device: &Device{},
			now:    sampleTS,
			want:   0,
			err:    errors.New("device time shift is not set"),
		},
		{
			name:   "Device time shift 1",
			device: &Device{timeShift: 1000 * sec},
			now:    sampleTS,
			want:   sampleTS - 1000*sec,
		},
		{
			name:   "Device time shift 2",
			device: &Device{timeShift: 1000 * hour},
			now:    sampleTS,
			want:   sampleTS - 1000*hour,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.device.TimeStamp(tt.now)
			h.AssertError(t, err, tt.err)
			if TimeStampDiff(got, tt.want) > sec {
				h.AssertEqual(t, got, tt.want)
			}
		})
	}
}

func TestDevice_Now(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   TimeStamp
		err    error
	}{
		{
			name:   "Device time shift is not set",
			device: &Device{},
			want:   0,
			err:    errors.New("device time shift is not set"),
		},
		{
			name:   "Device time shift 1",
			device: &Device{timeShift: 1000 * sec},
			want:   Now() - 1000*sec,
		},
		{
			name:   "Device time shift 2",
			device: &Device{timeShift: 1000 * hour},
			want:   Now() - 1000*hour,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.device.Now()
			h.AssertError(t, err, tt.err)
			if TimeStampDiff(got, tt.want) > 1*sec {
				h.AssertEqual(t, got, tt.want)
			}
		})
	}
}

func TestDevice_SetTimeShift(t *testing.T) {
	now := Now()
	tests := []struct {
		name    string
		device  *Device
		now     TimeStamp
		replyTS TimeStamp
		want    uint32
		err     error
	}{
		{
			name:    "error",
			device:  &Device{timeShift: 10 * hour},
			now:     now,
			replyTS: now + 1*sec,
			want:    uint32(10 * hour),
			err:     errors.New("device time cannot be in future"),
		},
		{
			name:    "success",
			device:  &Device{timeShift: 10 * hour},
			now:     now,
			replyTS: sampleTS,
			want:    uint32(now - sampleTS),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.device.SetTimeShift(tt.now, tt.replyTS)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, tt.device.timeShift, tt.want)
		})
	}
}

func TestDevice_Stage(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   DeviceStage
	}{
		{name: "Undiscovered", device: &Device{stage: Undiscovered}, want: Undiscovered},
		{name: "Found", device: &Device{stage: Found}, want: Found},
		{name: "Valid", device: &Device{stage: Valid}, want: Valid},
		{name: "Updated", device: &Device{stage: Updated}, want: Updated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.Stage()
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestDevice_SetStage(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		stage  DeviceStage
		want   DeviceStage
	}{
		{name: "Undiscovered", device: &Device{stage: Updated}, stage: Undiscovered, want: Undiscovered},
		{name: "Found", device: &Device{stage: Undiscovered}, stage: Found, want: Found},
		{name: "Valid", device: &Device{stage: Found}, stage: Valid, want: Valid},
		{name: "Updated", device: &Device{stage: Valid}, stage: Updated, want: Updated},
		{name: "Undiscovered 1", device: &Device{stage: Updated}, stage: -1, want: Undiscovered},
		{name: "Undiscovered 2", device: &Device{stage: Updated}, stage: 10, want: Undiscovered},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.device.SetStage(tt.stage)
			h.AssertEqual(t, DeviceStage(tt.device.stage), tt.want)
		})
	}
}

func TestDevice_FinalStage(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   DeviceStage
	}{
		{name: "Undiscovered", device: &Device{finalStage: Undiscovered}, want: Undiscovered},
		{name: "Found", device: &Device{finalStage: Found}, want: Found},
		{name: "Valid", device: &Device{finalStage: Valid}, want: Valid},
		{name: "Updated", device: &Device{finalStage: Updated}, want: Updated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.FinalStage()
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestDevice_SetFinalStage(t *testing.T) {
	tests := []struct {
		name       string
		device     *Device
		finalStage DeviceStage
		want       DeviceStage
	}{
		{name: "Undiscovered", device: &Device{finalStage: Updated}, finalStage: Undiscovered, want: Undiscovered},
		{name: "Found", device: &Device{finalStage: Undiscovered}, finalStage: Found, want: Found},
		{name: "Valid", device: &Device{finalStage: Found}, finalStage: Valid, want: Valid},
		{name: "Updated", device: &Device{finalStage: Valid}, finalStage: Updated, want: Updated},
		{name: "Undiscovered 1", device: &Device{finalStage: Updated}, finalStage: -1, want: Undiscovered},
		{name: "Undiscovered 2", device: &Device{finalStage: Updated}, finalStage: 10, want: Undiscovered},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.device.SetFinalStage(tt.finalStage)
			h.AssertEqual(t, DeviceStage(tt.device.finalStage), tt.want)
		})
	}
}

func TestDevice_InStage(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		stage  DeviceStage
		want   bool
	}{
		{name: "Undiscovered", device: &Device{stage: Updated}, stage: Undiscovered, want: true},
		{name: "Found", device: &Device{stage: Valid}, stage: Found, want: true},
		{name: "Valid", device: &Device{stage: Undiscovered}, stage: Valid, want: false},
		{name: "Updated", device: &Device{stage: Found}, stage: Updated, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.InStage(tt.stage)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestDevice_InFinalStage(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   bool
	}{
		{name: "Undiscovered", device: &Device{stage: Updated, finalStage: Undiscovered}, want: true},
		{name: "Found", device: &Device{stage: Valid, finalStage: Found}, want: true},
		{name: "Valid", device: &Device{stage: Undiscovered, finalStage: Valid}, want: false},
		{name: "Updated", device: &Device{stage: Found, finalStage: Updated}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.InFinalStage()
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestDevice_UpdatedAt(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   TimeStamp
	}{
		{
			name:   "Not set",
			device: &Device{},
			want:   0,
		},
		{
			name:   "1970-01-05 15:22:33",
			device: &Device{updatedAt: sampleTS},
			want:   sampleTS,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.UpdatedAt()
			if TimeStampDiff(got, tt.want) > 1*sec {
				h.AssertEqual(t, got, tt.want)
			}
		})
	}
}

func TestDevice_SetUpdatedNow(t *testing.T) {
	device := Device{updatedAt: sampleTS}
	h.AssertEqual(t, device.updatedAt, sampleTS)
	now := Now()
	device.SetUpdatedNow()
	if TimeStampDiff(device.updatedAt, now) > sec {
		h.AssertEqual(t, device.updatedAt, now)
	}
}

func TestDevice_UpdatedIn(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   TimeStamp
	}{
		{name: "A minute ago", device: &Device{updatedAt: Now() - 1*min}, want: 1 * min},
		{name: "In future", device: &Device{updatedAt: Now() + 1*min}, want: 0},
		{name: "updatedAt is not set", device: &Device{}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.UpdatedIn()
			if TimeStampDiff(got, tt.want) > 1*sec {
				h.AssertEqual(t, got, tt.want)
			}
		})
	}
}

func TestDevice_SetStateChangedNow(t *testing.T) {
	device := Device{stateChangedAt: sampleTS}
	h.AssertEqual(t, device.stateChangedAt, sampleTS)
	now := Now()
	device.SetStateChangedNow()
	if TimeStampDiff(device.stateChangedAt, now) > sec {
		h.AssertEqual(t, device.stateChangedAt, now)
	}
}

func TestDevice_SetStatePublishedNow(t *testing.T) {
	device := Device{statePublishedAt: sampleTS}
	h.AssertEqual(t, device.statePublishedAt, sampleTS)
	now := Now()
	device.SetStatePublishedNow()
	if TimeStampDiff(device.statePublishedAt, now) > sec {
		h.AssertEqual(t, device.statePublishedAt, now)
	}
}

func TestDevice_StateChangeUnpublished(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   bool
	}{
		{name: "state published before changed", device: &Device{stateChangedAt: sampleTS, statePublishedAt: sampleTS - 1*sec}, want: true},
		{name: "state published and changed same time", device: &Device{stateChangedAt: sampleTS, statePublishedAt: sampleTS}, want: false},
		{name: "state published after changed", device: &Device{stateChangedAt: sampleTS - 1*sec, statePublishedAt: sampleTS}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.StateChangeUnpublished()
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestDevices_Count(t *testing.T) {
	devices := Devices{
		1: &Device{stage: Undiscovered},
		2: &Device{stage: Found},
		3: &Device{stage: Valid},
		4: &Device{stage: Updated},
	}
	tests := []struct {
		name  string
		valid CheckDevice
		want  int
	}{
		{name: "None", valid: func(d *Device) bool { return false }, want: 0},
		{name: "All", valid: func(d *Device) bool { return true }, want: 4},
		{name: "Valid", valid: func(d *Device) bool { return d.Stage() >= Valid }, want: 2},
		{name: "Updated", valid: func(d *Device) bool { return d.Stage() >= Updated }, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := devices.Count(tt.valid)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestDevices_SetStage(t *testing.T) {
	devices := Devices{
		1: &Device{stage: Undiscovered},
		2: &Device{stage: Found},
		3: &Device{stage: Valid},
		4: &Device{stage: Updated},
	}
	tests := []struct {
		name      string
		stage     DeviceStage
		valid     CheckDevice
		wantCount int
	}{
		{name: "Valid to Updated", stage: Updated, valid: func(d *Device) bool { return d.Stage() == Valid }, wantCount: 2},
		{name: "All Found to Valid", stage: Valid, valid: func(d *Device) bool { return d.Stage() >= Found }, wantCount: 3},
		{name: "All to Found", stage: Found, valid: func(d *Device) bool { return true }, wantCount: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devices.SetStage(tt.stage, tt.valid)
			got := devices.Count(func(d *Device) bool { return d.Stage() == tt.stage })
			h.AssertEqual(t, got, tt.wantCount)
		})
	}
}

func Test_DeviceFound(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   bool
	}{
		{name: "Undiscovered", device: &Device{stage: Undiscovered}, want: false},
		{name: "Found", device: &Device{stage: Found}, want: true},
		{name: "Valid", device: &Device{stage: Valid}, want: true},
		{name: "Updated", device: &Device{stage: Updated}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeviceFound(tt.device)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_DeviceValid(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   bool
	}{
		{name: "Undiscovered", device: &Device{stage: Undiscovered}, want: false},
		{name: "Found", device: &Device{stage: Found}, want: false},
		{name: "Valid", device: &Device{stage: Valid}, want: true},
		{name: "Updated", device: &Device{stage: Updated}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeviceValid(tt.device)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_DeviceUpdated(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   bool
	}{
		{name: "Undiscovered", device: &Device{stage: Undiscovered}, want: false},
		{name: "Found", device: &Device{stage: Found}, want: false},
		{name: "Valid", device: &Device{stage: Valid}, want: false},
		{name: "Updated", device: &Device{stage: Updated}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeviceUpdated(tt.device)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_AnyDevice(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   bool
	}{
		{name: "Undiscovered", device: &Device{stage: Undiscovered}, want: true},
		{name: "Found", device: &Device{stage: Found}, want: true},
		{name: "Valid", device: &Device{stage: Valid}, want: true},
		{name: "Updated", device: &Device{stage: Updated}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AnyDevice(tt.device)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_DeviceNeedsUpdate(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   bool
	}{
		{name: "Undiscovered", device: &Device{stage: Undiscovered}, want: true},
		{name: "Found", device: &Device{stage: Found}, want: true},
		{name: "Valid", device: &Device{stage: Valid}, want: true},
		{name: "Updated", device: &Device{stage: Updated}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeviceNeedsUpdate(tt.device)
			h.AssertEqual(t, got, tt.want)
		})
	}
}
func Test_DeviceOutdated(t *testing.T) {
	logRe := regexp.MustCompile(`^\[INFO\]\s+outdated`)
	tests := []struct {
		name    string
		device  *Device
		timeout TimeStamp
		want    bool
		logRe   *regexp.Regexp
	}{
		{name: "Undiscovered", device: &Device{updatedAt: Now(), stage: Undiscovered}, timeout: min, want: false},
		{name: "Undiscovered timeout", device: &Device{updatedAt: Now() - 61*sec, stage: Undiscovered}, timeout: min, want: false},
		{name: "Found", device: &Device{updatedAt: Now(), stage: Found}, timeout: min, want: false},
		{name: "Found timeout", device: &Device{updatedAt: Now() - 61*sec, stage: Found}, timeout: min, want: true, logRe: logRe},
		{name: "Valid", device: &Device{updatedAt: Now(), stage: Valid}, timeout: min, want: false},
		{name: "Valid timeout", device: &Device{updatedAt: Now() - 61*sec, stage: Valid}, timeout: min, want: true, logRe: logRe},
		{name: "Updated", device: &Device{updatedAt: Now(), stage: Updated}, timeout: min, want: false},
		{name: "Updated timeout", device: &Device{updatedAt: Now() - 61*sec, stage: Updated}, timeout: min, want: true, logRe: logRe},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLog.Reset()
			got := DeviceOutdated(tt.timeout)(tt.device)
			h.AssertEqual(t, got, tt.want)
			h.AssertEqual(t, testLog.Message, tt.logRe)
		})
	}
}
