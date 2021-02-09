package miio

import (
	"bytes"
	"fmt"
	"testing"

	h "github.com/eip/miio2mqtt/helpers"
)

func Test_NewHelloPacket(t *testing.T) {
	want := &Packet{
		Magic:     0x2131,
		Length:    0x20,
		Unused:    0xffffffff,
		DeviceID:  0xffffffff,
		TimeStamp: 0xffffffff,
		Data:      []byte{},
	}
	copy(want.Checksum[:], h.FromHex("ffffffffffffffffffffffffffffffff"))
	got := NewHelloPacket()
	h.AssertEqual(t, got, want)
}

func Test_NewPacket(t *testing.T) {
	want := &Packet{
		Magic:     0x2131,
		Length:    0x33,
		Unused:    0x00,
		DeviceID:  0x00112233,
		TimeStamp: 0x00061e39,
		Data:      h.FromHex("31323334353637383940414243444546474849"),
	}
	copy(want.Checksum[:], h.FromHex("00000000000000000000000000000000"))
	got := NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849"))
	h.AssertEqual(t, got, want)
}

func Test_GetDeviceID(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want uint32
		err  error
	}{
		{
			name: "Hello Packet",
			data: h.FromHex("21310020ffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
			want: 0xffffffff,
		},
		{
			name: "Sample Packet",
			data: h.FromHex("21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849"),
			want: 0x00112233,
		},
		{
			name: "Short Packet",
			data: h.FromHex("21310033000000000011223300061e39"),
			err:  errInvalidDataLength,
		},
		{
			name: "Real Packet",
			data: h.FromHex("2131005000000000047bd1b55f53ee9bf0a2b109a80c902f0b55e5250e58f2cc95b21c4012d699586153e51f42d68c2ccdbb07b14326761acbe820ce8786ff4da71ff844841f7aee0c2b10c844b45245"),
			want: 0x047bd1b5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDeviceID(tt.data)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_decode(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want *Packet
		err  error
	}{
		{
			name: "Hello Packet",
			data: h.FromHex("21310020ffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
			want: NewHelloPacket(),
		},
		{
			name: "Sample Packet",
			data: h.FromHex("21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849"),
			want: NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
		},
		{
			name: "Short Packet",
			data: h.FromHex("21310033000000000011223300061e39"),
			err:  errInvalidDataLength,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decode(tt.data)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_Decode(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		token []byte
		want  *Packet
		err   error
	}{
		{
			name: "Hello Packet",
			data: h.FromHex("21310020ffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
			want: NewHelloPacket(),
		},
		{
			name: "Handshake Packet",
			data: h.FromHex("21310020000000000011223300061e3900000000000000000000000000000000"),
			want: NewPacket(0x00112233, sampleTS, nil),
		},
		{
			name:  "Sample Packet",
			data:  h.FromHex("21310040000000000011223300061e39b0cbb8837ed9a65a70165f2b7b4102722b487e7eed802b7df35c224caab8d216e43262c38b9cc073782c148668387d9e"),
			token: h.FromHex("00112233445566778899aabbccddeeff"),
			want:  NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
		},
		{
			name: "Invalid Packet (magic)",
			data: h.FromHex("22310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849"),
			err:  errInvalidMagicField,
		},
		{
			name: "Invalid Packet (length)",
			data: h.FromHex("21310032000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849"),
			err:  errInvalidDataLength,
		},
		{
			name: "Sample Packet (no token)",
			data: h.FromHex("21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849"),
			err:  errInvalidTokenLength,
		},
		{
			name:  "Invalid Packet (wrong checksum)",
			data:  h.FromHex("21310033000000000011223300061e3900749e5336e40d00b92fe648d67cef1031323334353637383940414243444546474849"),
			token: h.FromHex("00112233445566778899aabbccddeeff"),
			err:   errInvalidChecksum,
		},
		{
			name:  "Invalid Packet (wrong data length)",
			data:  h.FromHex("2131003f000000000011223300061e3996cf4e5b0b47cbe29244b3f4b899bbe32b487e7eed802b7df35c224caab8d216e43262c38b9cc073782c148668387d"),
			token: h.FromHex("00112233445566778899aabbccddeeff"),
			err:   errInvalidDataLength,
		},
		{
			name: "Short Packet",
			data: h.FromHex("21310033000000000011223300061e39"),
			err:  errInvalidDataLength,
		},
		{
			name:  "Real Packet 1",
			data:  h.FromHex("2131005000000000047bd1b55f53ee9bf0a2b109a80c902f0b55e5250e58f2cc95b21c4012d699586153e51f42d68c2ccdbb07b14326761acbe820ce8786ff4da71ff844841f7aee0c2b10c844b45245"),
			token: h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:  NewPacket(0x047bd1b5, 0x5f53ee9b, []byte(`{"method":"miIO.info","params":[],"id":1}`)),
		},
		{
			name:  "Real Packet 2",
			data:  h.FromHex("2131005000000000047bd1b5002feedece53f7b9e63ae50c3fc22fac87cc3ee7053510f79d4e36f4ff504d8da4391c467b067c3d5a777aca3ed402f9009821176bc6bffeb40994d5e6889e48836d54a6"),
			token: h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:  NewPacket(0x047bd1b5, 0x002feede, []byte(`{"result":["on","on",4,100,"off","on"],"id":1}`)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode(tt.data, tt.token)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestPacket_encode(t *testing.T) {
	tests := []struct {
		name     string
		packet   *Packet
		checksum []byte
		want     []byte
		err      error
	}{
		{
			name:   "Hello Packet",
			packet: NewHelloPacket(),
			want:   h.FromHex("21310020ffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
		},
		{
			name:   "Sample Packet (no checksum)",
			packet: NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
			want:   h.FromHex("21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849"),
		},
		{
			name:     "Sample Packet",
			packet:   NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
			checksum: h.FromHex("00112233445566778899aabbccddeeff"),
			want:     h.FromHex("21310033000000000011223300061e3900112233445566778899aabbccddeeff31323334353637383940414243444546474849"),
		},
		{
			name:     "Sample Packet (invalid checksum)",
			packet:   NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
			checksum: h.FromHex("0011223344556677"),
			err:      errInvalidChecksumLength,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.packet.encode(tt.checksum)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestPacket_Encode(t *testing.T) {
	tests := []struct {
		name   string
		packet *Packet
		token  []byte
		want   []byte
		err    error
	}{
		{
			name:   "Hello Packet",
			packet: NewHelloPacket(),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   h.FromHex("21310020ffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
		},
		{
			name:   "Handshake Packet",
			packet: NewPacket(0x00112233, sampleTS, nil),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   h.FromHex("21310020000000000011223300061e3900000000000000000000000000000000"),
		},
		{
			name:   "Sample Packet",
			packet: NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   h.FromHex("21310040000000000011223300061e39b0cbb8837ed9a65a70165f2b7b4102722b487e7eed802b7df35c224caab8d216e43262c38b9cc073782c148668387d9e"),
		},
		{
			name:   "Sample Packet (no token)",
			packet: NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
			want:   h.FromHex("21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849"),
		},
		{
			name:   "Real Packet 1",
			packet: NewPacket(0x047bd1b5, 0x002feede, []byte(`{"id":1,"method":"get_prop","params":["power","usb_state","aqi","battery","time_state","night_state"]}`)),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   h.FromHex("2131009000000000047bd1b5002feede54a30263b7b2015db6bdc3e7d5bf6853b952275b1e1fd9ed283c5ad34120d6982ccee490f5774502ee2833ecf7c8c178c01cb9250ee22edc72296cb393a9815dcb4c69e968271a25004626ead4c7abdd0332ddbccc48749ff1ddfe765439a06f6084ebdcca2ae9caeb2e755daaa5f3161cee3147f75a0f6ba4a127f89eb75eaa"),
		},
		{
			name:   "Real Packet 2",
			packet: NewPacket(0x047bd1b5, 0x002feede, []byte(`{"result":["on","on",4,100,"off","on"],"id":1}`)),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   h.FromHex("2131005000000000047bd1b5002feedece53f7b9e63ae50c3fc22fac87cc3ee7053510f79d4e36f4ff504d8da4391c467b067c3d5a777aca3ed402f9009821176bc6bffeb40994d5e6889e48836d54a6"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.packet.Encode(tt.token)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestPacket_CalcChecksum(t *testing.T) {
	tests := []struct {
		name   string
		packet *Packet
		token  []byte
		want   []byte
		err    error
	}{
		{
			name:   "Hello Packet",
			packet: NewHelloPacket(),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			err:    errInvalidDataLength,
		},
		{
			name:   "Handshake Packet",
			packet: NewPacket(0x00112233, sampleTS, nil),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			err:    errInvalidDataLength,
		},
		{
			name:   "Sample Packet 1",
			packet: NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   h.FromHex("dde071b5fa151be6c62a50d0274568f3"),
		},
		{
			name:   "Invalid Token",
			packet: NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
			token:  h.FromHex("00112233445566778899aabbccddee"),
			err:    errInvalidTokenLength,
		},
		{
			name:   "Real Packet 1",
			packet: packetFromHex("2131009000000000047bd1b5002feede54a30263b7b2015db6bdc3e7d5bf6853b952275b1e1fd9ed283c5ad34120d6982ccee490f5774502ee2833ecf7c8c178c01cb9250ee22edc72296cb393a9815dcb4c69e968271a25004626ead4c7abdd0332ddbccc48749ff1ddfe765439a06f6084ebdcca2ae9caeb2e755daaa5f3161cee3147f75a0f6ba4a127f89eb75eaa"),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   h.FromHex("54a30263b7b2015db6bdc3e7d5bf6853"),
		},
		{
			name:   "Real Packet 2",
			packet: packetFromHex("2131005000000000047bd1b5002feedece53f7b9e63ae50c3fc22fac87cc3ee7053510f79d4e36f4ff504d8da4391c467b067c3d5a777aca3ed402f9009821176bc6bffeb40994d5e6889e48836d54a6"),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   h.FromHex("ce53f7b9e63ae50c3fc22fac87cc3ee7"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.packet.CalcChecksum(tt.token)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestPacket_String(t *testing.T) {
	tests := []struct {
		name   string
		packet *Packet
		want   string
	}{
		{
			name:   "Hello Packet",
			packet: NewHelloPacket(),
			want:   "<Hello Packet>",
		},
		{
			name:   "Packet with no data",
			packet: NewPacket(0x00112233, sampleTS, nil),
			want:   `{deviceID:0x00112233,uptime:"111h22m33s"}`,
		},
		{
			name:   "Packet with string data",
			packet: NewPacket(0x00112233, sampleTS, []byte(`Hello, "World"`)),
			want:   `{deviceID:0x00112233,uptime:"111h22m33s",data:"Hello, \"World\""}`,
		},
		{
			name:   "Packet with json string data",
			packet: NewPacket(0x00112233, sampleTS, []byte(`{"id":1,"method":"miIO.info","params":[]}`)),
			want:   `{deviceID:0x00112233,uptime:"111h22m33s",data:{id:1,method:"miIO.info",params:[]}}`,
		},
		{
			name:   "Packet with binary data",
			packet: NewPacket(0x00112233, sampleTS, h.FromHex("0102030405060708090a0b0c0d0e0f10")),
			want:   `{deviceID:0x00112233,uptime:"111h22m33s",data:"0102030405060708090a0b0c0d0e0f10"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.packet.String()
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestPacket_Format(t *testing.T) {
	type want struct {
		s string
		v string
	}

	tests := []struct {
		name   string
		packet *Packet
		want   want
	}{
		{
			name:   "Hello Packet",
			packet: NewHelloPacket(),
			want:   want{s: "<Hello Packet>", v: "{Magic:2131 Length:0020 Unused:ffffffff DeviceID:ffffffff TimeStamp:1193046h28m15s Checksum:ffffffffffffffffffffffffffffffff}"},
		},
		{
			name:   "Packet with no data",
			packet: NewPacket(0x00112233, sampleTS, nil),
			want:   want{s: `{deviceID:0x00112233,uptime:"111h22m33s"}`, v: "{Magic:2131 Length:0020 Unused:00000000 DeviceID:00112233 TimeStamp:111h22m33s Checksum:00000000000000000000000000000000}"},
		},
		{
			name:   "Packet with string data",
			packet: NewPacket(0x00112233, sampleTS, []byte(`{"id":1,"method":"miIO.info","params":[]}`)),
			want:   want{s: `{deviceID:0x00112233,uptime:"111h22m33s",data:{id:1,method:"miIO.info",params:[]}}`, v: `{Magic:2131 Length:0049 Unused:00000000 DeviceID:00112233 TimeStamp:111h22m33s Checksum:00000000000000000000000000000000 Data:{"id":1,"method":"miIO.info","params":[]}}`},
		},
		{
			name:   "Packet with binary data",
			packet: NewPacket(0x00112233, sampleTS, h.FromHex("0102030405060708090a0b0c0d0e0f10")),
			want:   want{s: `{deviceID:0x00112233,uptime:"111h22m33s",data:"0102030405060708090a0b0c0d0e0f10"}`, v: "{Magic:2131 Length:0030 Unused:00000000 DeviceID:00112233 TimeStamp:111h22m33s Checksum:00000000000000000000000000000000 Data:0102030405060708090a0b0c0d0e0f10}"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h.AssertEqual(t, fmt.Sprintf("%s", tt.packet), tt.want.s)
			h.AssertEqual(t, fmt.Sprintf("%q", tt.packet), tt.want.s)
			h.AssertEqual(t, fmt.Sprintf("%v", tt.packet), tt.want.s)
			h.AssertEqual(t, fmt.Sprintf("%+v", tt.packet), tt.want.v)
		})
	}
}

func TestPacket_validateChecksum(t *testing.T) {
	tests := []struct {
		name   string
		packet *Packet
		token  []byte
		want   bool
		err    error
	}{
		{
			name:   "Hello Packet",
			packet: NewHelloPacket(),
			want:   true,
		},
		{
			name:   "Invalid Hello Packet",
			packet: packetFromHex("21310020ffffffffffffffffffffffffbdffffffffffffffffffffffffffffff"),
			want:   false,
		},
		{
			name:   "Handshake Packet",
			packet: NewPacket(0x00112233, sampleTS, nil),
			want:   true,
		},
		{
			name:   "Handshake Packet New",
			packet: packetFromHex("2131002000000000102695f0000b7567ffffffffffffffffffffffffffffffff"),
			want:   true,
		},
		{
			name:   "Invalid Handshake Packet",
			packet: packetFromHex("21310020000000000011223300061e390000bd00000000000000000000000000"),
			want:   false,
		},
		{
			name:   "Sample Packet",
			packet: packetFromHex("21310033000000000011223300061e39dde071b5fa151be6c62a50d0274568f331323334353637383940414243444546474849"),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   true,
		},
		{
			name:   "Sample Packet (no token)",
			packet: packetFromHex("21310033000000000011223300061e39dde071b5fa151be6c62a50d0274568f331323334353637383940414243444546474849"),
			err:    errInvalidTokenLength,
		},
		{
			name:   "Invalid Token",
			packet: packetFromHex("21310033000000000011223300061e39dde071b5fa151be6c62a50d0274568f331323334353637383940414243444546474849"),
			token:  h.FromHex("00112233445566778899aabbccddee"),
			err:    errInvalidTokenLength,
		},
		{
			name:   "Invalid Packet (0xff checksum)",
			packet: packetFromHex("21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849"),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   false,
		},
		{
			name:   "Invalid Packet (0x00 checksum)",
			packet: packetFromHex("21310033000000000011223300061e39ffffffffffffffffffffffffffffffff31323334353637383940414243444546474849"),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   false,
		},
		{
			name:   "Invalid Packet (wrong checksum)",
			packet: packetFromHex("21310033000000000011223300061e3900749e5336e40d00b92fe648d67cef1031323334353637383940414243444546474849"),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   false,
		},
		{
			name:   "Invalid Packet (no data)",
			packet: packetFromHex("21310033000000000011223300061e39dde071b5fa151be6c62a50d0274568f3"),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.packet.validateChecksum(tt.token)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestPacket_decrypt(t *testing.T) {
	tests := []struct {
		name   string
		packet *Packet
		token  []byte
		want   *Packet
		err    error
	}{
		{
			name:   "Hello Packet",
			packet: NewHelloPacket(),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   NewHelloPacket(),
		},
		{
			name:   "Handshake Packet",
			packet: NewPacket(0x00112233, sampleTS, nil),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   NewPacket(0x00112233, sampleTS, nil),
		},
		{
			name:   "Sample Packet (no token)",
			packet: NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
			want:   NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
		},
		{
			name:   "Sample Packet",
			packet: packetFromHex("21310040000000000011223300061e39b0cbb8837ed9a65a70165f2b7b4102722b487e7eed802b7df35c224caab8d216e43262c38b9cc073782c148668387d9e"),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
		},
		{
			name:   "Invalid Packet (wrong data length)",
			packet: packetFromHex("2131003f000000000011223300061e39fa0d07efa9bb95a3bdf5fe564e98ebe32b487e7eed802b7df35c224caab8d216e43262c38b9cc073782c148668387d"),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			err:    errInvalidDataLength,
		},
		{
			name:   "Invalid Token",
			packet: packetFromHex("21310040000000000011223300061e39b0cbb8837ed9a65a70165f2b7b4102722b487e7eed802b7df35c224caab8d216e43262c38b9cc073782c148668387d9e"),
			token:  h.FromHex("00112233445566778899aabbccddee"),
			err:    errInvalidTokenLength,
		},
		{
			name:   "Real Packet 1",
			packet: packetFromHex("2131009000000000047bd1b5002feede54a30263b7b2015db6bdc3e7d5bf6853b952275b1e1fd9ed283c5ad34120d6982ccee490f5774502ee2833ecf7c8c178c01cb9250ee22edc72296cb393a9815dcb4c69e968271a25004626ead4c7abdd0332ddbccc48749ff1ddfe765439a06f6084ebdcca2ae9caeb2e755daaa5f3161cee3147f75a0f6ba4a127f89eb75eaa"),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   NewPacket(0x047bd1b5, 0x002feede, []byte(`{"id":1,"method":"get_prop","params":["power","usb_state","aqi","battery","time_state","night_state"]}`)),
		},
		{
			name:   "Real Packet 2",
			packet: packetFromHex("2131005000000000047bd1b5002feedece53f7b9e63ae50c3fc22fac87cc3ee7053510f79d4e36f4ff504d8da4391c467b067c3d5a777aca3ed402f9009821176bc6bffeb40994d5e6889e48836d54a6"),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   NewPacket(0x047bd1b5, 0x002feede, []byte(`{"result":["on","on",4,100,"off","on"],"id":1}`)),
		},
		{
			name:   "Real Packet 3",
			packet: packetFromHex("2131006000000000047bd1b5002feee7106d8a634882f4bafc8b8a2681b8314eae1c63b7536cc3329f54668849a278897790b94fe65f5effc0d332eb2f10e4e709ce88217b477bc9b56afaf48e9dd38e9c1f77ed444ae71e96c03a88a151a8c5"),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   NewPacket(0x047bd1b5, 0x002feee7, []byte(`{"id":2,"method":"set_time_state","params":["on"]}`)),
		},
		{
			name:   "Real Packet 4",
			packet: packetFromHex("2131004000000000047bd1b5002feee7e3697b20842094f6132854dcde69f3059115aa3c9b13ca32129ce4b02b2dbdd9ab107a52bde2821fd7a642ad47598527"),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   NewPacket(0x047bd1b5, 0x002feee7, []byte(`{"result":["ok"],"id":2}`)),
		},
		{
			name:   "Real Packet 5",
			packet: packetFromHex("2131006000000000047bd1b5002feeecd526f789bd5f6309d54da10e150e583470dade107aea785a733025d970048d018f50166b851b22edcfae5bb8e8994b3e1f7f6799d4667efef67ab371a4fbbe6b5e1fa2252a605da3e82b2e4693ca2c0f"),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   NewPacket(0x047bd1b5, 0x002feeec, []byte(`{"id":3,"method":"set_time_state","params":["off"]}`)),
		},
		{
			name:   "Real Packet 6",
			packet: packetFromHex("2131004000000000047bd1b5002feeec77c05d0bfa755624dc61d65b5a7de7fe9115aa3c9b13ca32129ce4b02b2dbdd93c71b6cb99f0a2e9a12e2558124ba9d0"),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   NewPacket(0x047bd1b5, 0x002feeec, []byte(`{"result":["ok"],"id":3}`)),
		},
		{
			name:   "Real Packet 7",
			packet: packetFromHex("2131008000000000047bd1b5002feeefc15580d0ce026d1ca125662ae96e50d508c5bcac739f16daa9b251e7c114318f2255ba8d655f6bf57a841b028797229bdce38a9b27756ec1bc4624c0066fd6f3453d0484e88aad434d44a2278b1e3b125a1653c55f816eed9758be3bdb16a39f05c5a6b3cb19e57cbf2c7cea804123f2"),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   NewPacket(0x047bd1b5, 0x002feeef, []byte(`{"id":4,"method":"get_prop","params":["night_state","night_beg_time","night_end_time"]}`)),
		},
		{
			name:   "Real Packet 8",
			packet: packetFromHex("2131005000000000047bd1b5002feeef12500f47c724baaf8e7600bffd837701053510f79d4e36f4ff504d8da4391c4620051270d7c1fb142f1bd3968a92997c2717c809fb1605db6db2a9227b49bde3"),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   NewPacket(0x047bd1b5, 0x002feeef, []byte(`{"result":["on",72000,32400],"id":4}`)),
		},
		{
			name:   "Real Packet 9",
			packet: packetFromHex("2131009000000000047bd1b5002feef3aad3c9950d72802aba392cf27f40245d91c188c2e3657a17374b5befc36b9f59dd6e1dc0493abd2046ee0ad26bac0443e21af255927d332075a4e92582f96132a475b877e1ab5f1d5139afd9fd8a4f3d4029cce8ed80911577fae0cbe55c9ac95c007ec9097f5b63190375f4b0433893e5a85d4d7674ab3b4420cb344816e533"),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   NewPacket(0x047bd1b5, 0x002feef3, []byte(`{"id":5,"method":"get_prop","params":["power","usb_state","aqi","battery","time_state","night_state"]}`)),
		},
		{
			name:   "Real Packet 10",
			packet: packetFromHex("2131005000000000047bd1b5002feef4ebce1f825f30f4c8e0e5c6e06122f908053510f79d4e36f4ff504d8da4391c466d7b9560e6b73c7d21f14a8e36e4b31fe0b865fa91b4818a2a1264932896bf17"),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   NewPacket(0x047bd1b5, 0x002feef4, []byte(`{"result":["on","on",3,100,"off","on"],"id":5}`)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.packet.decrypt(tt.token)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestPacket_encrypt(t *testing.T) {
	tests := []struct {
		name   string
		packet *Packet
		token  []byte
		want   *Packet
		err    error
	}{
		{
			name:   "Hello Packet",
			packet: NewHelloPacket(),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   NewHelloPacket(),
		},
		{
			name:   "Handshake Packet",
			packet: NewPacket(0x00112233, sampleTS, nil),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   NewPacket(0x00112233, sampleTS, nil),
		},
		{
			name:   "Sample Packet (no token)",
			packet: NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
			want:   NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
		},
		{
			name:   "Sample Packet",
			packet: NewPacket(0x00112233, sampleTS, h.FromHex("31323334353637383940414243444546474849")),
			token:  h.FromHex("00112233445566778899aabbccddeeff"),
			want:   packetFromHex("21310040000000000011223300061e39b0cbb8837ed9a65a70165f2b7b4102722b487e7eed802b7df35c224caab8d216e43262c38b9cc073782c148668387d9e"),
		},
		{
			name:   "Real Packet 1",
			packet: NewPacket(0x047bd1b5, 0x002feede, []byte(`{"id":1,"method":"get_prop","params":["power","usb_state","aqi","battery","time_state","night_state"]}`)),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   packetFromHex("2131009000000000047bd1b5002feede54a30263b7b2015db6bdc3e7d5bf6853b952275b1e1fd9ed283c5ad34120d6982ccee490f5774502ee2833ecf7c8c178c01cb9250ee22edc72296cb393a9815dcb4c69e968271a25004626ead4c7abdd0332ddbccc48749ff1ddfe765439a06f6084ebdcca2ae9caeb2e755daaa5f3161cee3147f75a0f6ba4a127f89eb75eaa"),
		},
		{
			name:   "Real Packet 2",
			packet: NewPacket(0x047bd1b5, 0x002feede, []byte(`{"result":["on","on",4,100,"off","on"],"id":1}`)),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   packetFromHex("2131005000000000047bd1b5002feedece53f7b9e63ae50c3fc22fac87cc3ee7053510f79d4e36f4ff504d8da4391c467b067c3d5a777aca3ed402f9009821176bc6bffeb40994d5e6889e48836d54a6"),
		},
		{
			name:   "Real Packet 3",
			packet: NewPacket(0x047bd1b5, 0x002feee7, []byte(`{"id":2,"method":"set_time_state","params":["on"]}`)),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   packetFromHex("2131006000000000047bd1b5002feee7106d8a634882f4bafc8b8a2681b8314eae1c63b7536cc3329f54668849a278897790b94fe65f5effc0d332eb2f10e4e709ce88217b477bc9b56afaf48e9dd38e9c1f77ed444ae71e96c03a88a151a8c5"),
		},
		{
			name:   "Real Packet 4",
			packet: NewPacket(0x047bd1b5, 0x002feee7, []byte(`{"result":["ok"],"id":2}`)),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   packetFromHex("2131004000000000047bd1b5002feee7e3697b20842094f6132854dcde69f3059115aa3c9b13ca32129ce4b02b2dbdd9ab107a52bde2821fd7a642ad47598527"),
		},
		{
			name:   "Real Packet 5",
			packet: NewPacket(0x047bd1b5, 0x002feeec, []byte(`{"id":3,"method":"set_time_state","params":["off"]}`)),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   packetFromHex("2131006000000000047bd1b5002feeecd526f789bd5f6309d54da10e150e583470dade107aea785a733025d970048d018f50166b851b22edcfae5bb8e8994b3e1f7f6799d4667efef67ab371a4fbbe6b5e1fa2252a605da3e82b2e4693ca2c0f"),
		},
		{
			name:   "Real Packet 6",
			packet: NewPacket(0x047bd1b5, 0x002feeec, []byte(`{"result":["ok"],"id":3}`)),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   packetFromHex("2131004000000000047bd1b5002feeec77c05d0bfa755624dc61d65b5a7de7fe9115aa3c9b13ca32129ce4b02b2dbdd93c71b6cb99f0a2e9a12e2558124ba9d0"),
		},
		{
			name:   "Real Packet 7",
			packet: NewPacket(0x047bd1b5, 0x002feeef, []byte(`{"id":4,"method":"get_prop","params":["night_state","night_beg_time","night_end_time"]}`)),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   packetFromHex("2131008000000000047bd1b5002feeefc15580d0ce026d1ca125662ae96e50d508c5bcac739f16daa9b251e7c114318f2255ba8d655f6bf57a841b028797229bdce38a9b27756ec1bc4624c0066fd6f3453d0484e88aad434d44a2278b1e3b125a1653c55f816eed9758be3bdb16a39f05c5a6b3cb19e57cbf2c7cea804123f2"),
		},
		{
			name:   "Real Packet 8",
			packet: NewPacket(0x047bd1b5, 0x002feeef, []byte(`{"result":["on",72000,32400],"id":4}`)),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   packetFromHex("2131005000000000047bd1b5002feeef12500f47c724baaf8e7600bffd837701053510f79d4e36f4ff504d8da4391c4620051270d7c1fb142f1bd3968a92997c2717c809fb1605db6db2a9227b49bde3"),
		},
		{
			name:   "Real Packet 9",
			packet: NewPacket(0x047bd1b5, 0x002feef3, []byte(`{"id":5,"method":"get_prop","params":["power","usb_state","aqi","battery","time_state","night_state"]}`)),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   packetFromHex("2131009000000000047bd1b5002feef3aad3c9950d72802aba392cf27f40245d91c188c2e3657a17374b5befc36b9f59dd6e1dc0493abd2046ee0ad26bac0443e21af255927d332075a4e92582f96132a475b877e1ab5f1d5139afd9fd8a4f3d4029cce8ed80911577fae0cbe55c9ac95c007ec9097f5b63190375f4b0433893e5a85d4d7674ab3b4420cb344816e533"),
		},
		{
			name:   "Real Packet 10",
			packet: NewPacket(0x047bd1b5, 0x002feef4, []byte(`{"result":["on","on",3,100,"off","on"],"id":5}`)),
			token:  h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:   packetFromHex("2131005000000000047bd1b5002feef4ebce1f825f30f4c8e0e5c6e06122f908053510f79d4e36f4ff504d8da4391c466d7b9560e6b73c7d21f14a8e36e4b31fe0b865fa91b4818a2a1264932896bf17"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.packet.encrypt(tt.token)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestPacketData_String(t *testing.T) {
	tests := []struct {
		name string
		data Payload
		want string
	}{
		{name: "nil", data: nil, want: ""},
		{name: "empty slice", data: nil, want: ""},
		{name: "string", data: Payload(`Hello, "World"`), want: `Hello, "World"`},
		{name: "json string", data: Payload(`{"id":1,"method":"miIO.info","params":[]}`), want: `{id:1,method:"miIO.info",params:[]}`},
		{name: "binary", data: h.FromHex("0102030405060708090a0b0c0d0e0f10"), want: "0102030405060708090a0b0c0d0e0f10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.data.String()
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func TestPacketData_string(t *testing.T) {
	type args struct {
		quotes   bool
		simplify bool
	}
	tests := []struct {
		name string
		data Payload
		args args
		want string
	}{
		{name: "nil", data: nil, args: args{quotes: false, simplify: false}, want: ""},
		{name: "nil quoted", data: nil, args: args{quotes: true, simplify: false}, want: "\"\""},
		{name: "nil simplified", data: nil, args: args{quotes: false, simplify: true}, want: ""},
		{name: "nil quoted simplified", data: nil, args: args{quotes: true, simplify: true}, want: "\"\""},
		{name: "empty slice", data: nil, args: args{quotes: false, simplify: false}, want: ""},
		{name: "empty slice quoted", data: nil, args: args{quotes: true, simplify: false}, want: "\"\""},
		{name: "empty slice simplified", data: nil, args: args{quotes: false, simplify: true}, want: ""},
		{name: "empty slice quoted simplified", data: nil, args: args{quotes: true, simplify: true}, want: "\"\""},
		{name: "string", data: Payload(`Hello, "World"`), args: args{quotes: false, simplify: false}, want: `Hello, "World"`},
		{name: "string quoted", data: Payload(`Hello, "World"`), args: args{quotes: true, simplify: false}, want: `"Hello, \"World\""`},
		{name: "string simplified", data: Payload(`Hello, "World"`), args: args{quotes: false, simplify: true}, want: `Hello, "World"`},
		{name: "string quoted simplified", data: Payload(`Hello, "World"`), args: args{quotes: true, simplify: true}, want: `"Hello, \"World\""`},
		{name: "json string", data: Payload(`{"id":1,"method":"miIO.info","params":[]}`), args: args{quotes: false, simplify: false}, want: `{"id":1,"method":"miIO.info","params":[]}`},
		{name: "json string quoted", data: Payload(`{"id":1,"method":"miIO.info","params":[]}`), args: args{quotes: true, simplify: false}, want: `{"id":1,"method":"miIO.info","params":[]}`},
		{name: "json string simplified", data: Payload(`{"id":1,"method":"miIO.info","params":[]}`), args: args{quotes: false, simplify: true}, want: `{id:1,method:"miIO.info",params:[]}`},
		{name: "json string quoted simplified", data: Payload(`{"id":1,"method":"miIO.info","params":[]}`), args: args{quotes: true, simplify: true}, want: `{id:1,method:"miIO.info",params:[]}`},
		{name: "binary", data: h.FromHex("0102030405060708090a0b0c0d0e0f10"), args: args{quotes: false, simplify: false}, want: "0102030405060708090a0b0c0d0e0f10"},
		{name: "binary quoted", data: h.FromHex("0102030405060708090a0b0c0d0e0f10"), args: args{quotes: true, simplify: false}, want: "\"0102030405060708090a0b0c0d0e0f10\""},
		{name: "binary simplified", data: h.FromHex("0102030405060708090a0b0c0d0e0f10"), args: args{quotes: false, simplify: true}, want: "0102030405060708090a0b0c0d0e0f10"},
		{name: "binary quoted simplified", data: h.FromHex("0102030405060708090a0b0c0d0e0f10"), args: args{quotes: true, simplify: true}, want: "\"0102030405060708090a0b0c0d0e0f10\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.data.string(tt.args.quotes, tt.args.simplify)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_dataLen(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
		want int
	}{
		{
			name: "Non-empty data",
			data: []byte{1, 2, 3, 4, 5},
			want: 5,
		},
		{
			name: "Empty data",
			data: []byte{},
			want: 0,
		},
		{
			name: "Nil data",
			data: nil,
			want: 0,
		},
		{
			name: "Integer data",
			data: 123,
			want: -1,
		},
		{
			name: "String data",
			data: "Hello",
			want: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dataLen(tt.data)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_pkcs7pad(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		blockSize int
		want      []byte
		err       error
	}{
		{
			name:      "Small block size",
			data:      []byte{1, 2, 3, 4, 5},
			blockSize: 1,
			err:       errInvalidBlockSize,
		},
		{
			name:      "Large block size",
			data:      []byte{1, 2, 3, 4, 5},
			blockSize: 256,
			err:       errInvalidBlockSize,
		},
		{
			name:      "No data",
			data:      []byte{},
			blockSize: 16,
			err:       errInvalidDataLength,
		},
		{
			name:      "1 byte padding",
			data:      []byte{1, 2, 3, 4, 5, 6, 7},
			blockSize: 2,
			want:      []byte{1, 2, 3, 4, 5, 6, 7, 1},
		},
		{
			name:      "2 byte padding",
			data:      []byte{1, 2, 3, 4, 5, 6},
			blockSize: 2,
			want:      []byte{1, 2, 3, 4, 5, 6, 2, 2},
		},
		{
			name:      "3 byte padding",
			data:      []byte{1, 2, 3, 4, 5},
			blockSize: 8,
			want:      []byte{1, 2, 3, 4, 5, 3, 3, 3},
		},
		{
			name:      "7 byte padding",
			data:      []byte{1},
			blockSize: 8,
			want:      []byte{1, 7, 7, 7, 7, 7, 7, 7},
		},
		{
			name:      "8 byte padding",
			data:      []byte{1, 2, 3, 4, 5, 6, 7, 8},
			blockSize: 8,
			want:      []byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 8, 8, 8, 8, 8, 8, 8},
		},
		{
			name:      "255 byte padding",
			data:      bytes.Repeat([]byte{48}, 255),
			blockSize: 255,
			want:      append(bytes.Repeat([]byte{48}, 255), bytes.Repeat([]byte{255}, 255)...),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pkcs7pad(tt.data, tt.blockSize)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func Test_pkcs7strip(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		blockSize int
		want      []byte
		err       error
	}{
		{
			name:      "Small block size",
			data:      []byte{1, 2, 3, 4, 5},
			blockSize: 1,
			err:       errInvalidBlockSize,
		},
		{
			name:      "Large block size",
			data:      []byte{1, 2, 3, 4, 5},
			blockSize: 256,
			err:       errInvalidBlockSize,
		},
		{
			name:      "No data",
			data:      []byte{},
			blockSize: 16,
			err:       errInvalidDataLength,
		},
		{
			name:      "invalid padding",
			data:      []byte{1, 2, 3, 4, 5, 3, 2, 3},
			blockSize: 8,
			err:       errInvalidPadding,
		},
		{
			name:      "1 byte padding",
			data:      []byte{1, 2, 3, 4, 5, 6, 7, 1},
			blockSize: 2,
			want:      []byte{1, 2, 3, 4, 5, 6, 7},
		},
		{
			name:      "2 byte padding",
			data:      []byte{1, 2, 3, 4, 5, 6, 2, 2},
			blockSize: 2,
			want:      []byte{1, 2, 3, 4, 5, 6},
		},
		{
			name:      "3 byte padding",
			data:      []byte{1, 2, 3, 4, 5, 3, 3, 3},
			blockSize: 8,
			want:      []byte{1, 2, 3, 4, 5},
		},
		{
			name:      "7 byte padding",
			data:      []byte{1, 7, 7, 7, 7, 7, 7, 7},
			blockSize: 8,
			want:      []byte{1},
		},
		{
			name:      "8 byte padding",
			data:      []byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 8, 8, 8, 8, 8, 8, 8},
			blockSize: 8,
			want:      []byte{1, 2, 3, 4, 5, 6, 7, 8},
		},
		{
			name:      "255 byte padding",
			data:      append(bytes.Repeat([]byte{48}, 255), bytes.Repeat([]byte{255}, 255)...),
			blockSize: 255,
			want:      bytes.Repeat([]byte{48}, 255),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pkcs7strip(tt.data, tt.blockSize)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
		})
	}
}

func packetFromHex(s string) *Packet {
	p, err := decode(h.FromHex(s))
	if err != nil {
		return nil
	}
	return p
}

func Test_FailDecode(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		token []byte
		want  *Packet
		err   error
	}{
		// {
		// 	name: "Fail Hello Packet",
		// 	data: h.FromHex("21310020ffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
		// 	want: NewHelloPacket(),
		// },
		{
			name:  "Fail Real Packet",
			data:  h.FromHex("2131005000000000047bd1b5002feedece53f7b9e63ae50c3fc22fac87cc3ee7053510f79d4e36f4ff504d8da4391c467b067c3d5a777aca3ed402f9009821176bc6bffeb40994d5e6889e48836d54a6"),
			token: h.FromHex("9c3b2d1da5beceee2808a3d3653b485d"),
			want:  NewPacket(0x047bd1b5, 0x002feede, []byte(`{"xresult":["on","on",4,100,"off","on"],"id":1}`)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode(tt.data, tt.token)
			h.AssertError(t, err, tt.err)
			h.AssertEqual(t, got, tt.want)
			// fmt.Printf("%%s:  %s\n", got)
			// fmt.Printf("%%q:  %q\n", got)
			// fmt.Printf("%%v:  %v\n", got)
			// fmt.Printf("%%+v: %+v\n", got)
			// fmt.Printf("%%#v: %#v\n", got)
		})
	}
}
