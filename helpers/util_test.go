package helpers

import (
	"testing"

	log "github.com/go-pkgz/lgr"
)

func Test_FromHex(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want []byte
	}{
		{
			name: "Sample hex string",
			arg:  "000102030405060708090A0B0C0D0E0F",
			want: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f},
		},
		{
			name: "Invalid hex string 1",
			arg:  "000102030405060708090A0B0C0D0E0FF",
			want: nil,
		},
		{
			name: "Invalid hex string 2",
			arg:  "_000102030405060708090A0B0C0D0E0F",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromHex(tt.arg)
			AssertEqual(t, got, tt.want)
		})
	}
}

func Test_IsPrintableASCII(t *testing.T) {
	tests := []struct {
		name string
		arg  []byte
		want bool
	}{
		{
			name: "Printable slice",
			arg:  []byte{0x20, 0x21, 0x22, 0x23, 0x7c, 0x7d, 0x7e, 0x7f},
			want: true,
		},
		{
			name: "Non-printable slice 1",
			arg:  []byte{0x20, 0x21, 0x22, 0x23, 0x1f, 0x7c, 0x7d, 0x7e, 0x7f},
			want: false,
		},
		{
			name: "Non-printable slice 2",
			arg:  []byte{0x20, 0x21, 0x22, 0x23, 0x80, 0x7c, 0x7d, 0x7e, 0x7f},
			want: false,
		},
		{
			name: "Empty slice",
			arg:  []byte{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPrintableASCII(tt.arg)
			AssertEqual(t, got, tt.want)
		})
	}
}

func Test_IsJSON(t *testing.T) {
	tests := []struct {
		name string
		arg  []byte
		want bool
	}{
		{name: "nil", arg: nil, want: false},
		{name: "empty slice", arg: []byte{}, want: false},
		{name: "not JSON string 1", arg: []byte(`Hello, "World"`), want: false},
		{name: "not JSON string 2", arg: []byte(`{Hello, "World"}`), want: false},
		{name: "not JSON string 3", arg: []byte(`{Hello: "World"}`), want: false},
		{name: "JSON string 1", arg: []byte(`{"method":"get_prop","params":["power","usb_state","aqi","battery"],"id":2}`), want: true},
		{name: "JSON string 2", arg: []byte(`{"RESULT":["on","on",20,100],"ID":2}`), want: true},
		{name: "JSON string 3", arg: []byte(`{"fw_ver":"1.4.3_8103","hw_ver":"MW300"}`), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsJSON(tt.arg)
			AssertEqual(t, got, tt.want)
		})
	}
}

func Test_StripJSONQuotes(t *testing.T) {
	tests := []struct {
		name string
		arg  []byte
		want []byte
	}{
		{name: "nil", arg: nil, want: nil},
		{name: "empty slice", arg: []byte{}, want: nil},
		{name: "JSON string 1", arg: []byte(`{"method":"get_prop","params":["power","usb_state","aqi","battery"],"id":2}`), want: []byte(`{method:"get_prop",params:["power","usb_state","aqi","battery"],id:2}`)},
		{name: "JSON string 2", arg: []byte(`{"RESULT":["on","on",20,100],"ID":2}`), want: []byte(`{RESULT:["on","on",20,100],ID:2}`)},
		{name: "JSON string 3", arg: []byte(`{"fw_ver":"1.4.3_8103","hw_ver":"MW300"}`), want: []byte(`{fw_ver:"1.4.3_8103",hw_ver:"MW300"}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripJSONQuotes(tt.arg)
			AssertEqual(t, got, tt.want)
		})
	}
}

func TestTestLog_Write(t *testing.T) {
	type spy struct {
		message string
	}
	tests := []struct {
		name string
		log  *TestLog
		arg  []byte
		want int
		err  error
		spy  spy
	}{
		{
			name: "nil data",
			log:  &TestLog{Message: "foo"},
			arg:  nil,
			want: 0,
			spy:  spy{message: ""},
		},
		{
			name: "Empty data",
			log:  &TestLog{Message: "foo"},
			arg:  []byte(""),
			want: 0,
			spy:  spy{message: ""},
		},
		{
			name: "Sample string",
			log:  &TestLog{Message: "foo"},
			arg:  []byte("bar!"),
			want: 4,
			spy:  spy{message: "bar!"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.log.Write(tt.arg)
			AssertError(t, err, tt.err)
			AssertEqual(t, got, tt.want)
			AssertEqual(t, tt.log.Message, tt.spy.message)
		})
	}
}

func TestTestLog_Reset(t *testing.T) {
	log := &TestLog{Message: "foo"}
	AssertEqual(t, log.Message, "foo")
	log.Reset()
	AssertEqual(t, log.Message, "")
}

func Test_InitTestLog(t *testing.T) {
	logger := InitTestLog()
	AssertEqual(t, logger, &TestLog{})
	log.Print("[DEBUG] Foo bar")
	AssertEqual(t, logger.Message, "")
	log.Print("[INFO] Foo bar")
	AssertEqual(t, logger.Message, "[INFO]  Foo bar\n")
	log.Print("[ERROR] Foo bar")
	AssertEqual(t, logger.Message, "[ERROR] Foo bar\n")
}
