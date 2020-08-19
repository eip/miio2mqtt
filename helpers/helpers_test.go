package helpers

import (
	"errors"
	"fmt"
	"testing"
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

func Test_formatValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{name: "int", value: int(237), want: "237"},
		{name: "int8", value: int8(111), want: "111"},
		{name: "int16", value: int16(237), want: "237"},
		{name: "int32", value: int32(237), want: "237"},
		{name: "int64", value: int64(237), want: "237"},
		{name: "byte", value: byte(237), want: "ed"},
		{name: "uint8", value: uint8(237), want: "ed"},
		{name: "uint16", value: uint16(237), want: "00ed"},
		{name: "uint32", value: uint32(237), want: "000000ed"},
		{name: "uint64", value: uint64(237), want: "00000000000000ed"},
		{name: "bool true", value: bool(true), want: "true"},
		{name: "bool false", value: bool(false), want: "false"},
		{
			name:  "[]byte hex",
			value: FromHex("21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849"),
			want:  "21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849",
		},
		{
			name:  "[]byte ascii",
			value: []byte("Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor."),
			want:  "\"Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor.\"",
		},
		{name: "[]byte nil", value: []byte(nil), want: "[]byte(nil)"},
		{name: "string 1", value: string("foo bar"), want: "\"foo bar\""},
		{name: "string 2", value: string("Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor."), want: "\"Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor.\""},
		{name: "struct 1", value: struct{}{}, want: "{}"},
		{name: "struct 2", value: struct {
			foo string
			bar int
			baz []byte
		}{"foo", 237, FromHex("3132333435")}, want: "{foo:foo bar:237 baz:[49 50 51 52 53]}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatValue(tt.value); got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func Test_formatError(t *testing.T) {
	tests := []struct {
		name    string
		argGot  string
		argWant string
		want    string
	}{
		{name: "short strings", argGot: "foo bar", argWant: "baz qux", want: "got foo bar, want baz qux"},
		{name: "long strings", argGot: "ut pulvinar nisl eu eros", argWant: "cras at molestie orci ac", want: "\ngot  ut pulvinar nisl eu eros\nwant cras at molestie orci ac"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatError(tt.argGot, tt.argWant); got != tt.want {
				t.Errorf("\ngot  %q\nwant %q", got, tt.want)
			}
		})
	}
}

type mockT struct {
	helperCalls int
	errorCalls  int
	fatalfCalls int
	message     string
}

func (mt *mockT) Helper() {
	mt.helperCalls++
}

func (mt *mockT) Error(args ...interface{}) {
	mt.errorCalls++
	mt.message = fmt.Sprint(args...)
}

func (mt *mockT) Fatalf(format string, args ...interface{}) {
	mt.fatalfCalls++
	mt.message = fmt.Sprintf(format, args...)
}

func Test_AssertEqual(t *testing.T) {
	tests := []struct {
		name        string
		argGot      interface{}
		argWant     interface{}
		helperCalls int
		errorCalls  int
		want        string
	}{
		{name: "same values", argGot: int(237), argWant: int(237), helperCalls: 1, errorCalls: 0, want: ""},
		{name: "different values", argGot: int(237), argWant: int(0), helperCalls: 1, errorCalls: 1, want: "got 237, want 0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockT{}
			AssertEqual(mt, tt.argGot, tt.argWant)
			if mt.helperCalls != tt.helperCalls {
				t.Errorf("Helper() called %d times, want %d", mt.helperCalls, tt.helperCalls)
			}
			if mt.errorCalls != tt.errorCalls {
				t.Errorf("Error() called %d times, want %d", mt.errorCalls, tt.errorCalls)
			}
			if mt.message != tt.want {
				t.Errorf("\ngot  %q\nwant %q", mt.message, tt.want)
			}
		})
	}
}

func Test_AssertError(t *testing.T) {
	sampleErr := errors.New("sample error")
	tests := []struct {
		name        string
		errGot      error
		errWant     error
		helperCalls int
		fatalfCalls int
		want        string
	}{
		{name: "no errors", errGot: nil, errWant: nil, helperCalls: 1, fatalfCalls: 0, want: ""},
		{name: "same errors", errGot: sampleErr, errWant: sampleErr, helperCalls: 1, fatalfCalls: 0, want: ""},
		{name: "same error messages", errGot: sampleErr, errWant: errors.New("sample error"), helperCalls: 1, fatalfCalls: 0, want: ""},
		{name: "nil and error", errGot: nil, errWant: sampleErr, helperCalls: 1, fatalfCalls: 1, want: "expected to get an error: \"sample error\""},
		{name: "error and nil", errGot: sampleErr, errWant: nil, helperCalls: 1, fatalfCalls: 1, want: "got unexpected error: \"sample error\""},
		{name: "errors", errGot: sampleErr, errWant: errors.New("another error"), helperCalls: 1, fatalfCalls: 1, want: "\ngot error \"sample error\", want \"another error\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockT{}
			AssertError(mt, tt.errGot, tt.errWant)
			if mt.helperCalls != tt.helperCalls {
				t.Errorf("Helper() called %d times, want %d", mt.helperCalls, tt.helperCalls)
			}
			if mt.fatalfCalls != tt.fatalfCalls {
				t.Errorf("Fatalf() called %d times, want %d", mt.fatalfCalls, tt.fatalfCalls)
			}
			if mt.message != tt.want {
				t.Errorf("\ngot  %q\nwant %q", mt.message, tt.want)
			}
		})
	}
}
