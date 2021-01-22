package helpers

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"time"
)

type T interface {
	Helper()
	Error(args ...interface{})
	Fatalf(format string, args ...interface{})
}

func AssertEqual(t T, got, want interface{}) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	t.Error(formatError(formatValue(got), formatValue(want)))
}

func AssertError(t T, got, want error) {
	t.Helper()
	if got == nil && want != nil {
		t.Fatalf("expected to get an error: %q", want)
		return
	}
	if got != nil && want == nil {
		t.Fatalf("got unexpected error: %q", got)
		return
	}
	if got == want {
		return
	}
	if got.Error() != want.Error() {
		t.Fatalf("\ngot error %q, want %q", got, want)
	}
}

func FromHex(s string) []byte {
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	return bytes
}

func IsPrintableASCII(b []byte) bool {
	if len(b) < 1 {
		return false
	}
	const min byte = 0x20
	const max byte = 0x7f
	for i := 0; i < len(b); i++ {
		if b[i] < min || b[i] > max {
			return false
		}
	}
	return true
}

func TimeDiff(t1, t2 time.Time) time.Duration {
	diff := t1.Sub(t2)
	if diff < 0 {
		diff = -diff
	}
	return time.Duration(diff)
}

func TimeStampDiff(t1, t2 int64) time.Duration {
	diff := t1 - t2
	if diff < 0 {
		diff = -diff
	}
	return time.Duration(diff)
}

func formatValue(value interface{}) string {
	v := reflect.ValueOf(value)
	k := v.Kind()
	switch k {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		if v.IsNil() {
			return fmt.Sprintf("%#v", value)
		}
	}
	if k == reflect.Ptr {
		v = v.Elem()
		k = v.Kind()
	}
	switch k {
	case reflect.Bool:
		return fmt.Sprintf("%v", value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", value)
	case reflect.Uint8:
		return fmt.Sprintf("%02x", value)
	case reflect.Uint16:
		return fmt.Sprintf("%04x", value)
	case reflect.Uint32:
		return fmt.Sprintf("%08x", value)
	case reflect.Uint64:
		return fmt.Sprintf("%016x", value)
	case reflect.Slice:
		switch value := value.(type) {
		case []byte:
			if IsPrintableASCII(value) {
				return fmt.Sprintf("%q", value)
			}
			return fmt.Sprintf("%x", value)
		default:
			fmt.Printf("##### need support for %s (%T)\n", k, value)
			return fmt.Sprintf("%#v", value)
		}
	case reflect.String:
		return fmt.Sprintf("%q", value)
	case reflect.Struct:
		return fmt.Sprintf("%+v", value)
	default:
		fmt.Printf("##### need support for %s (%T)\n", k, value)
		return fmt.Sprintf("%#v", value)
	}
}

func formatError(got, want string) string {
	if len(got)+len(want) < 40 {
		return fmt.Sprintf("got %s, want %s", got, want)
	}
	return fmt.Sprintf("\ngot  %s\nwant %s", got, want)
}
