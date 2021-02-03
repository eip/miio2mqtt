package helpers

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"
)

type stringer int64

func (ts stringer) String() string {
	return fmt.Sprintf("%#016X", int64(ts))
}

func Test_formatValue(t *testing.T) {
	tests := []struct {
		name string
		arg  interface{}
		want string
	}{
		{name: "int", arg: int(237), want: "237"},
		{name: "*int", arg: func(v int) *int { return &v }(237), want: "237"},
		{name: "int8", arg: int8(111), want: "111"},
		{name: "int16", arg: int16(237), want: "237"},
		{name: "int32", arg: int32(237), want: "237"},
		{name: "int64", arg: int64(237), want: "237"},
		{name: "*int64", arg: func(v int64) *int64 { return &v }(237), want: "237"},
		{name: "byte", arg: byte(237), want: "ed"},
		{name: "uint8", arg: uint8(237), want: "ed"},
		{name: "uint16", arg: uint16(237), want: "00ed"},
		{name: "uint32", arg: uint32(237), want: "000000ed"},
		{name: "uint64", arg: uint64(237), want: "00000000000000ed"},
		{name: "*uint64", arg: func(v uint64) *uint64 { return &v }(237), want: "00000000000000ed"},
		{name: "bool true", arg: bool(true), want: "true"},
		{name: "*bool true", arg: func(v bool) *bool { return &v }(true), want: "true"},
		{name: "bool false", arg: bool(false), want: "false"},
		{name: "Duration", arg: time.Duration(11*3600+22*60+33) * time.Second, want: "11h22m33s"},
		{name: "stringer", arg: stringer(237), want: "0X00000000000000ED"},
		{
			name: "[]byte hex",
			arg:  FromHex("21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849"),
			want: "21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849",
		},
		{
			name: "*[]byte hex",
			arg:  func(v []byte) *[]byte { return &v }(FromHex("21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849")),
			want: "21310033000000000011223300061e390000000000000000000000000000000031323334353637383940414243444546474849",
		},
		{
			name: "[]byte ascii",
			arg:  []byte("Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor."), // cspell: disable-line
			want: "\"Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor.\"",     // cspell: disable-line
		},
		{
			name: "*[]byte ascii",
			arg:  func(v []byte) *[]byte { return &v }([]byte("Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor.")), // cspell: disable-line
			want: "\"Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor.\"",                                           // cspell: disable-line
		},
		{name: "[]byte nil", arg: []byte(nil), want: "[]byte(nil)"},
		{name: "string 1", arg: string("foo bar"), want: "\"foo bar\""},
		{name: "string 2", arg: string("Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor."), want: "\"Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor.\""},                                // cspell: disable-line
		{name: "*string 2", arg: func(v string) *string { return &v }("Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor."), want: "\"Vivamus sed gravida nulla, id luctus nulla. Nunc at tempor.\""}, // cspell: disable-line
		{name: "struct 1", arg: struct{}{}, want: "{}"},
		{name: "struct 2", arg: struct {
			foo string
			bar int
			baz []byte
		}{"foo", 237, FromHex("3132333435")}, want: "{foo:foo bar:237 baz:[49 50 51 52 53]}"},
		{name: "*struct 2", arg: &struct {
			foo string
			bar int
			baz []byte
		}{"foo", 237, FromHex("3132333435")}, want: "{foo:foo bar:237 baz:[49 50 51 52 53]}"},
		{name: "*Regexp", arg: regexp.MustCompile("^Foo \\d+$"), want: "\"^Foo \\\\d+$\""},
		{name: "[]int", arg: []int{1, 2, 3, 4, 5}, want: "[]int{1, 2, 3, 4, 5}"},
		{name: "float64", arg: float64(237.89), want: "237.89"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatValue(tt.arg); got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func Test_formatError(t *testing.T) {
	type args struct {
		got  string
		want string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "Short strings", args: args{got: "foo bar", want: "baz qux"}, want: "got foo bar, want baz qux"},
		{name: "Long strings", args: args{got: "ut pulvinar nisl eu eros", want: "cras at molestie orci ac"}, want: "\ngot  ut pulvinar nisl eu eros\nwant cras at molestie orci ac"}, // cspell: disable-line
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatError(tt.args.got, tt.args.want); got != tt.want {
				t.Errorf("\ngot  %q\nwant %q", got, tt.want)
			}
		})
	}
}

func Test_matchString(t *testing.T) {
	type args struct {
		got interface{}
		re  interface{}
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "Empty string and Regexp", args: args{got: string(""), re: regexp.Regexp{}}, want: true},
		{name: "Empty string and nil *Regexp", args: args{got: string(""), re: &regexp.Regexp{}}, want: true},
		{name: "Empty string and *Regexp", args: args{got: string(""), re: regexp.MustCompile("")}, want: true},
		{name: "Sample string matching Regexp", args: args{got: string("Foo 237"), re: *regexp.MustCompile("^Foo \\d+$")}, want: true},
		{name: "Sample string matching *Regexp", args: args{got: string("Foo 237"), re: regexp.MustCompile("^Foo \\d+$")}, want: true},
		{name: "Sample string not matching Regexp", args: args{got: string("Foo 237x"), re: *regexp.MustCompile("^Foo \\d+$")}, want: false},
		{name: "Sample string not matching *Regexp", args: args{got: string("Foo 237x"), re: regexp.MustCompile("^Foo \\d+$")}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchString(tt.args.got, tt.args.re); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

type mockT struct {
	testing.T
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
	type args struct {
		got  interface{}
		want interface{}
	}
	type spy struct {
		helperCalls int
		errorCalls  int
	}
	tests := []struct {
		name string
		args args
		want string
		spy  spy
	}{
		{name: "Same values", args: args{got: int(237), want: int(237)}, want: "", spy: spy{helperCalls: 1, errorCalls: 0}},
		{name: "Different values", args: args{got: int(237), want: int(0)}, want: "got 237, want 0", spy: spy{helperCalls: 1, errorCalls: 1}},
		{name: "Matching string", args: args{got: string("Foo 237"), want: regexp.MustCompile("^Foo \\d+$")}, want: "", spy: spy{helperCalls: 1, errorCalls: 0}},
		{name: "Not matching string", args: args{got: string("Foo 237x"), want: regexp.MustCompile("^Foo \\d+$")}, want: "got \"Foo 237x\", want \"^Foo \\\\d+$\"", spy: spy{helperCalls: 1, errorCalls: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockT{}
			AssertEqual(mt, tt.args.got, tt.args.want)
			if mt.helperCalls != tt.spy.helperCalls {
				t.Errorf("Helper() called %d times, want %d", mt.helperCalls, tt.spy.helperCalls)
			}
			if mt.errorCalls != tt.spy.errorCalls {
				t.Errorf("Error() called %d times, want %d", mt.errorCalls, tt.spy.errorCalls)
			}
			if mt.message != tt.want {
				t.Errorf("\ngot  %q\nwant %q", mt.message, tt.want)
			}
		})
	}
}

func Test_AssertError(t *testing.T) {
	type args struct {
		got  error
		want error
	}
	type spy struct {
		helperCalls int
		fatalfCalls int
	}
	sampleErr := errors.New("sample error")
	tests := []struct {
		name string
		args args
		want string
		spy  spy
	}{
		{name: "No errors", args: args{got: nil, want: nil}, want: "", spy: spy{helperCalls: 1, fatalfCalls: 0}},
		{name: "Same errors", args: args{got: sampleErr, want: sampleErr}, want: "", spy: spy{helperCalls: 1, fatalfCalls: 0}},
		{name: "Same error messages", args: args{got: sampleErr, want: errors.New("sample error")}, want: "", spy: spy{helperCalls: 1, fatalfCalls: 0}},
		{name: "Nil and error", args: args{got: nil, want: sampleErr}, want: "expected to get an error: \"sample error\"", spy: spy{helperCalls: 1, fatalfCalls: 1}},
		{name: "Error and nil", args: args{got: sampleErr, want: nil}, want: "got unexpected error: \"sample error\"", spy: spy{helperCalls: 1, fatalfCalls: 1}},
		{name: "Errors", args: args{got: sampleErr, want: errors.New("another error")}, want: "\ngot error \"sample error\", want \"another error\"", spy: spy{helperCalls: 1, fatalfCalls: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &mockT{}
			AssertError(mt, tt.args.got, tt.args.want)
			if mt.helperCalls != tt.spy.helperCalls {
				t.Errorf("Helper() called %d times, want %d", mt.helperCalls, tt.spy.helperCalls)
			}
			if mt.fatalfCalls != tt.spy.fatalfCalls {
				t.Errorf("Fatalf() called %d times, want %d", mt.fatalfCalls, tt.spy.fatalfCalls)
			}
			if mt.message != tt.want {
				t.Errorf("\ngot  %q\nwant %q", mt.message, tt.want)
			}
		})
	}
}
