package helpers

import (
	"testing"

	log "github.com/go-pkgz/lgr"
)

func Test_FromHex(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []byte
	}{
		{
			name: "Sample hex string",
			data: "000102030405060708090A0B0C0D0E0F",
			want: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f},
		},
		{
			name: "Invalid hex string 1",
			data: "000102030405060708090A0B0C0D0E0FF",
			want: nil,
		},
		{
			name: "Invalid hex string 2",
			data: "_000102030405060708090A0B0C0D0E0F",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromHex(tt.data)
			AssertEqual(t, got, tt.want)
		})
	}
}

func Test_IsPrintableASCII(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "Printable slice",
			data: []byte{0x20, 0x21, 0x22, 0x23, 0x7c, 0x7d, 0x7e, 0x7f},
			want: true,
		},
		{
			name: "Non-printable slice 1",
			data: []byte{0x20, 0x21, 0x22, 0x23, 0x1f, 0x7c, 0x7d, 0x7e, 0x7f},
			want: false,
		},
		{
			name: "Non-printable slice 2",
			data: []byte{0x20, 0x21, 0x22, 0x23, 0x80, 0x7c, 0x7d, 0x7e, 0x7f},
			want: false,
		},
		{
			name: "Empty slice",
			data: []byte{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPrintableASCII(tt.data)
			AssertEqual(t, got, tt.want)
		})
	}
}

func TestTestLog_Write(t *testing.T) {
	tests := []struct {
		name        string
		log         *TestLog
		p           []byte
		want        int
		wantMessage string
		err         error
	}{
		{
			name:        "nil data",
			log:         &TestLog{Message: "foo"},
			p:           nil,
			want:        0,
			wantMessage: "",
		},
		{
			name:        "Empty data",
			log:         &TestLog{Message: "foo"},
			p:           []byte(""),
			want:        0,
			wantMessage: "",
		},
		{
			name:        "Sample string",
			log:         &TestLog{Message: "foo"},
			p:           []byte("bar!"),
			want:        4,
			wantMessage: "bar!",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.log.Write(tt.p)
			AssertError(t, err, tt.err)
			AssertEqual(t, got, tt.want)
			AssertEqual(t, tt.log.Message, tt.wantMessage)
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
