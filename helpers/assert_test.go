package helpers

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
)

func Test_formatValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{name: "int", value: int(237), want: "237"},
		{name: "*int", value: func(v int) *int { return &v }(237), want: "237"},
		{name: "int8", value: int8(111), want: "111"},
		{name: "int16", value: int16(237), want: "237"},
		{name: "int32", value: int32(237), want: "237"},
		{name: "int64", value: int64(237), want: "237"},
		{name: "*int64", value: func(v int64) *int64 { return &v }(237), want: "237"},
		{name: "byte", value: byte(237), want: "ed"},
		{name: "uint8", value: uint8(237), want: "ed"},
		{name: "uint16", value: uint16(237), want: "00ed"},
		{name: "uint32", value: uint32(237), want: "000000ed"},
		{name: "uint64", value: uint64(237), want: "00000000000000ed"},
		{name: "*uint64", value: func(v uint64) *uint64 { return &v }(237), want: "00000000000000ed"},
		{name: "bool true", value: bool(true), want: "true"},
		{name: "*bool true", value: func(v bool) *bool { return &v }(true), want: "true"},
		{name: "bool false", value: bool(false), want: "false"},
		{
			name:  "[]byte hex",
			value: FromHex("21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849"),
			want:  "21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849",
		},
		{
			name:  "*[]byte hex",
			value: func(v []byte) *[]byte { return &v }(FromHex("21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849")),
			want:  "21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849",
		},
		{
			name:  "[]byte ascii",
			value: []byte("Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor."), // cspell: disable-line
			want:  "\"Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor.\"",     // cspell: disable-line
		},
		{
			name:  "*[]byte ascii",
			value: func(v []byte) *[]byte { return &v }([]byte("Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor.")), // cspell: disable-line
			want:  "\"Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor.\"",                                           // cspell: disable-line
		},
		{name: "[]byte nil", value: []byte(nil), want: "[]byte(nil)"},
		{name: "string 1", value: string("foo bar"), want: "\"foo bar\""},
		{name: "string 2", value: string("Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor."), want: "\"Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor.\""},                                // cspell: disable-line
		{name: "*string 2", value: func(v string) *string { return &v }("Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor."), want: "\"Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor.\""}, // cspell: disable-line
		{name: "struct 1", value: struct{}{}, want: "{}"},
		{name: "struct 2", value: struct {
			foo string
			bar int
			baz []byte
		}{"foo", 237, FromHex("3132333435")}, want: "{foo:foo bar:237 baz:[49 50 51 52 53]}"},
		{name: "*struct 2", value: &struct {
			foo string
			bar int
			baz []byte
		}{"foo", 237, FromHex("3132333435")}, want: "{foo:foo bar:237 baz:[49 50 51 52 53]}"},
		{name: "*Regexp", value: regexp.MustCompile("^Foo \\d+$"), want: "\"^Foo \\\\d+$\""},
		{name: "[]int", value: []int{1, 2, 3, 4, 5}, want: "[]int{1, 2, 3, 4, 5}"},
		{name: "float64", value: float64(237.89), want: "237.89"},
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
		{name: "Short strings", argGot: "foo bar", argWant: "baz qux", want: "got foo bar, want baz qux"},
		{name: "Long strings", argGot: "ut pulvinar nisl eu eros", argWant: "cras at molestie orci ac", want: "\ngot  ut pulvinar nisl eu eros\nwant cras at molestie orci ac"}, // cspell: disable-line
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatError(tt.argGot, tt.argWant); got != tt.want {
				t.Errorf("\ngot  %q\nwant %q", got, tt.want)
			}
		})
	}
}

func Test_matchString(t *testing.T) {
	tests := []struct {
		name   string
		argGot interface{}
		argRe  interface{}
		want   bool
	}{
		{name: "Empty string and Regexp", argGot: string(""), argRe: regexp.Regexp{}, want: true},
		{name: "Empty string and nil *Regexp", argGot: string(""), argRe: &regexp.Regexp{}, want: true},
		{name: "Empty string and *Regexp", argGot: string(""), argRe: regexp.MustCompile(""), want: true},
		{name: "Sample string matching Regexp", argGot: string("Foo 237"), argRe: *regexp.MustCompile("^Foo \\d+$"), want: true},
		{name: "Sample string matching *Regexp", argGot: string("Foo 237"), argRe: regexp.MustCompile("^Foo \\d+$"), want: true},
		{name: "Sample string not matching Regexp", argGot: string("Foo 237x"), argRe: *regexp.MustCompile("^Foo \\d+$"), want: false},
		{name: "Sample string not matching *Regexp", argGot: string("Foo 237x"), argRe: regexp.MustCompile("^Foo \\d+$"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchString(tt.argGot, tt.argRe); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
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
		{name: "Same values", argGot: int(237), argWant: int(237), helperCalls: 1, errorCalls: 0, want: ""},
		{name: "Different values", argGot: int(237), argWant: int(0), helperCalls: 1, errorCalls: 1, want: "got 237, want 0"},
		{name: "Matching string", argGot: string("Foo 237"), argWant: regexp.MustCompile("^Foo \\d+$"), helperCalls: 1, errorCalls: 0, want: ""},
		{name: "Not matching string", argGot: string("Foo 237x"), argWant: regexp.MustCompile("^Foo \\d+$"), helperCalls: 1, errorCalls: 1, want: "got \"Foo 237x\", want \"^Foo \\\\d+$\""},
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
		{name: "No errors", errGot: nil, errWant: nil, helperCalls: 1, fatalfCalls: 0, want: ""},
		{name: "Same errors", errGot: sampleErr, errWant: sampleErr, helperCalls: 1, fatalfCalls: 0, want: ""},
		{name: "Same error messages", errGot: sampleErr, errWant: errors.New("sample error"), helperCalls: 1, fatalfCalls: 0, want: ""},
		{name: "Nil and error", errGot: nil, errWant: sampleErr, helperCalls: 1, fatalfCalls: 1, want: "expected to get an error: \"sample error\""},
		{name: "Error and nil", errGot: sampleErr, errWant: nil, helperCalls: 1, fatalfCalls: 1, want: "got unexpected error: \"sample error\""},
		{name: "Errors", errGot: sampleErr, errWant: errors.New("another error"), helperCalls: 1, fatalfCalls: 1, want: "\ngot error \"sample error\", want \"another error\""},
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
