package miio

import (
	"errors"
	"testing"
	"time"

	h "github.com/eip/miio2mqtt/helpers"
)

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

func TestDevice_Now(t *testing.T) {
	tests := []struct {
		name   string
		device Device
		want   time.Time
		err    error
	}{
		{
			name:   "Device time shift is not set",
			device: Device{},
			want:   time.Unix(0, 0),
			err:    errors.New("device time shift is not set"),
		},
		{
			name:   "Device time shift 1",
			device: Device{TimeShift: 1000 * time.Hour},
			want:   time.Now().Add(-1000 * time.Hour),
		},
		{
			name:   "Device time shift 2",
			device: Device{TimeShift: -1000 * time.Hour},
			want:   time.Now().Add(1000 * time.Hour),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.device.Now()
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_deviceRequest(t *testing.T) {
	tests := []struct {
		name       string
		data       string
		deviceID   uint32
		requestID  uint32
		deviceTime time.Time
		token      []byte
		wantPkt    *Packet
		wantData   []byte
		err        error
	}{
		{
			name:       "Sample Request",
			data:       "123456789@ABCDEFGHI",
			deviceID:   0x00112233,
			requestID:  0,
			deviceTime: sampleTime,
			token:      h.FromHex("00112233445566778899aabbccddeeff"),
			wantPkt:    NewPacket(0x00112233, sampleTime, h.FromHex("31323334353637383940414243444546474849")),
			wantData:   h.FromHex("21310040000000000011223300061e39b0cbb8837ed9a65a70165f2b7b4102722b487e7eed802b7df35c224caab8d216e43262c38b9cc073782c148668387d9e"),
		},
		{
			name:       "Invalid Token",
			data:       "123456789@ABCDEFGHI",
			deviceID:   0x00112233,
			requestID:  0,
			deviceTime: sampleTime,
			token:      h.FromHex("00112233445566778899aabbccddeeff00"),
			wantPkt:    NewPacket(0x00112233, sampleTime, h.FromHex("31323334353637383940414243444546474849")),
			err:        errInvalidTokenLength,
		},
		{
			name:       "Real Request",
			data:       `{"method":"miIO.info","params":[],"id":#}`,
			deviceID:   0x00112233,
			requestID:  123,
			deviceTime: sampleTime,
			token:      h.FromHex("00112233445566778899aabbccddeeff"),
			wantPkt:    NewPacket(0x00112233, sampleTime, []byte(`{"method":"miIO.info","params":[],"id":123}`)),
			wantData:   h.FromHex("21310050000000000011223300061e39bc379b48c96b52ffd80dcbd9153594d12f42719f20d1969cd734b11bee043ad5a740d19c6e38ff8438a641c565d7b6f68c0c7008b88bc6869531a7ceac7818e2"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPkt, gotData, err := deviceRequest([]byte(tt.data), tt.deviceID, tt.requestID, tt.deviceTime, tt.token)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, gotPkt, tt.wantPkt)
			// h.AssertEqual(t, gotPkt.Data, tt.wantPkt.Data)
			h.AssertEqual(t, gotData, tt.wantData)
		})
	}
}

func TestDevice_Request(t *testing.T) {
	tests := []struct {
		name          string
		device        Device
		data          string
		wantRequestID uint32
		wantPktData   string
		wantPktTime   time.Time
		err           error
	}{
		{
			name:   "Device time shift is not set",
			device: Device{},
			err:    errors.New("device time shift is not set"),
		},
		{
			name:          "Real Request",
			device:        Device{DeviceCfg: DeviceCfg{ID: 0x00112233}, Token: [16]byte{}, TimeShift: 1000 * time.Hour, requestID: 122},
			data:          `{"method":"miIO.info","params":[],"id":#}`,
			wantRequestID: 123,
			wantPktData:   `{"method":"miIO.info","params":[],"id":123}`,
			wantPktTime:   time.Now().Add(-1000 * time.Hour),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPkt, _, err := tt.device.Request([]byte(tt.data))
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, tt.device.requestID, tt.wantRequestID)
			if gotPkt != nil {
				h.AssertEqual(t, gotPkt.Data, []byte(tt.wantPktData))
				gotPktTime := time.Unix(int64(gotPkt.Stamp), 0)
				if pktTimeDiff := tt.wantPktTime.Sub(gotPktTime); pktTimeDiff < -500*time.Millisecond || pktTimeDiff > 1500*time.Millisecond { // ignore 2 second difference
					h.AssertEqual(t, gotPktTime, tt.wantPktTime)
				}
			} else if err == nil {
				h.AssertEqual(t, gotPkt, NewPacket(tt.device.ID, tt.wantPktTime, []byte(tt.data)))
			}
		})
	}
}

func TestDevice_GetStage(t *testing.T) {
	tests := []struct {
		name   string
		device Device
		want   DeviceStage
	}{
		{name: "Undiscovered", device: Device{stage: int32(Undiscovered)}, want: Undiscovered},
		{name: "Found", device: Device{stage: int32(Found)}, want: Found},
		{name: "Valid", device: Device{stage: int32(Valid)}, want: Valid},
		{name: "Updated", device: Device{stage: int32(Updated)}, want: Updated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.GetStage()
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestDevice_SetStage(t *testing.T) {
	tests := []struct {
		name   string
		device Device
		stage  DeviceStage
		want   DeviceStage
	}{
		{name: "Undiscovered", device: Device{stage: int32(Updated)}, stage: Undiscovered, want: Undiscovered},
		{name: "Found", device: Device{stage: int32(Undiscovered)}, stage: Found, want: Found},
		{name: "Valid", device: Device{stage: int32(Found)}, stage: Valid, want: Valid},
		{name: "Updated", device: Device{stage: int32(Valid)}, stage: Updated, want: Updated},
		{name: "Undiscovered 1", device: Device{stage: int32(Updated)}, stage: -1, want: Undiscovered},
		{name: "Undiscovered 2", device: Device{stage: int32(Updated)}, stage: 10, want: Undiscovered},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.device.SetStage(tt.stage)
			h.AssertEqual(t, DeviceStage(tt.device.stage), tt.want)
		})
	}
}

func TestDevices_Count(t *testing.T) {
	devices := Devices{
		1: &Device{stage: int32(Undiscovered)},
		2: &Device{stage: int32(Found)},
		3: &Device{stage: int32(Valid)},
		4: &Device{stage: int32(Updated)},
	}
	tests := []struct {
		name  string
		valid CheckDevice
		want  int
	}{
		{name: "None", valid: func(d *Device) bool { return false }, want: 0},
		{name: "All", valid: func(d *Device) bool { return true }, want: 4},
		{name: "Valid", valid: func(d *Device) bool { return d.GetStage() >= Valid }, want: 2},
		{name: "Updated", valid: func(d *Device) bool { return d.GetStage() >= Updated }, want: 1},
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
		1: &Device{stage: int32(Undiscovered)},
		2: &Device{stage: int32(Found)},
		3: &Device{stage: int32(Valid)},
		4: &Device{stage: int32(Updated)},
	}
	tests := []struct {
		name      string
		stage     DeviceStage
		valid     CheckDevice
		wantCount int
	}{
		{name: "Valid to Updated", stage: Updated, valid: func(d *Device) bool { return d.GetStage() == Valid }, wantCount: 2},
		{name: "All Found to Valid", stage: Valid, valid: func(d *Device) bool { return d.GetStage() >= Found }, wantCount: 3},
		{name: "All to Found", stage: Found, valid: func(d *Device) bool { return true }, wantCount: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devices.SetStage(tt.stage, tt.valid)
			got := devices.Count(func(d *Device) bool { return d.GetStage() == tt.stage })
			h.AssertEqual(t, got, tt.wantCount)
		})
	}
}

func Test_DeviceFound(t *testing.T) {
	tests := []struct {
		name   string
		device Device
		want   bool
	}{
		{name: "Undiscovered", device: Device{stage: int32(Undiscovered)}, want: false},
		{name: "Found", device: Device{stage: int32(Found)}, want: true},
		{name: "Valid", device: Device{stage: int32(Valid)}, want: true},
		{name: "Updated", device: Device{stage: int32(Updated)}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeviceFound(&tt.device)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_DeviceValid(t *testing.T) {
	tests := []struct {
		name   string
		device Device
		want   bool
	}{
		{name: "Undiscovered", device: Device{stage: int32(Undiscovered)}, want: false},
		{name: "Found", device: Device{stage: int32(Found)}, want: false},
		{name: "Valid", device: Device{stage: int32(Valid)}, want: true},
		{name: "Updated", device: Device{stage: int32(Updated)}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeviceValid(&tt.device)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_DeviceUpdated(t *testing.T) {
	tests := []struct {
		name   string
		device Device
		want   bool
	}{
		{name: "Undiscovered", device: Device{stage: int32(Undiscovered)}, want: false},
		{name: "Found", device: Device{stage: int32(Found)}, want: false},
		{name: "Valid", device: Device{stage: int32(Valid)}, want: false},
		{name: "Updated", device: Device{stage: int32(Updated)}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeviceUpdated(&tt.device)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_AnyDevice(t *testing.T) {
	tests := []struct {
		name   string
		device Device
		want   bool
	}{
		{name: "Undiscovered", device: Device{stage: int32(Undiscovered)}, want: true},
		{name: "Found", device: Device{stage: int32(Found)}, want: true},
		{name: "Valid", device: Device{stage: int32(Valid)}, want: true},
		{name: "Updated", device: Device{stage: int32(Updated)}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AnyDevice(&tt.device)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_DeviceNeedsUpdate(t *testing.T) {
	tests := []struct {
		name   string
		device Device
		want   bool
	}{
		{name: "Undiscovered", device: Device{stage: int32(Undiscovered)}, want: true},
		{name: "Found", device: Device{stage: int32(Found)}, want: true},
		{name: "Valid", device: Device{stage: int32(Valid)}, want: true},
		{name: "Updated", device: Device{stage: int32(Updated)}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeviceNeedsUpdate(&tt.device)
			h.AssertEqual(t, got, tt.want)
		})
	}
}
func Test_DeviceOutdated(t *testing.T) {
	tests := []struct {
		name    string
		device  Device
		timeout time.Duration
		want    bool
	}{
		{name: "Undiscovered", device: Device{UpdatedAt: time.Now(), stage: int32(Undiscovered)}, timeout: time.Minute, want: false},
		{name: "Undiscovered timeout", device: Device{UpdatedAt: time.Now().Add(-61 * time.Second), stage: int32(Undiscovered)}, timeout: time.Minute, want: false},
		{name: "Found", device: Device{UpdatedAt: time.Now(), stage: int32(Found)}, timeout: time.Minute, want: false},
		{name: "Found timeout", device: Device{UpdatedAt: time.Now().Add(-61 * time.Second), stage: int32(Found)}, timeout: time.Minute, want: true},
		{name: "Valid", device: Device{UpdatedAt: time.Now(), stage: int32(Valid)}, timeout: time.Minute, want: false},
		{name: "Valid timeout", device: Device{UpdatedAt: time.Now().Add(-61 * time.Second), stage: int32(Valid)}, timeout: time.Minute, want: true},
		{name: "Updated", device: Device{UpdatedAt: time.Now(), stage: int32(Updated)}, timeout: time.Minute, want: false},
		{name: "Updated timeout", device: Device{UpdatedAt: time.Now().Add(-61 * time.Second), stage: int32(Updated)}, timeout: time.Minute, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeviceOutdated(tt.timeout)(&tt.device)
			h.AssertEqual(t, got, tt.want)
		})
	}
}
