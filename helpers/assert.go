package helpers

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"
)

// AssertEqual asserts that two objects are equal.
func AssertEqual(t testing.TB, got, want interface{}) {
	t.Helper()
	if matchString(got, want) || reflect.DeepEqual(got, want) {
		return
	}
	t.Error(formatError(formatValue(got), formatValue(want)))
}

// AssertError asserts that two error objects are both nil or equal or contain equal messages.
func AssertError(t testing.TB, got, want error) {
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

func matchString(got, re interface{}) bool {
	switch re := re.(type) {
	case regexp.Regexp:
		got := fmt.Sprint(got)
		return (len(re.String()) == 0 && len(got) == 0) || (len(re.String()) > 0 && re.MatchString(got))
	case *regexp.Regexp:
		got := fmt.Sprint(got)
		return ((re == nil || len(re.String()) == 0) && len(got) == 0) || (re != nil && len(re.String()) > 0 && re.MatchString(got))
	default:
		return false
	}
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
	switch value := value.(type) {
	case fmt.Stringer:
		return value.String()
	case fmt.Formatter:
		return fmt.Sprint(value)
	}
	if k == reflect.Ptr {
		v = v.Elem()
		k = v.Kind()
	}
	value = v.Interface()
	switch value := value.(type) {
	case fmt.Stringer:
		return value.String()
	case fmt.Formatter:
		return fmt.Sprint(value)
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
			fmt.Printf("### helpers.formatValue(): need support for %s (%T)\n", k, value)
			return fmt.Sprintf("%#v", value)
		}
	case reflect.String:
		return fmt.Sprintf("%q", value)
	case reflect.Struct:
		switch value := value.(type) {
		case regexp.Regexp:
			return fmt.Sprintf("%q", &value)
		default:
			return fmt.Sprintf("%+v", value)
		}
	default:
		fmt.Printf("### helpers.formatValue(): need support for %s (%T)\n", k, value)
		return fmt.Sprintf("%#v", value)
	}
}

func formatError(got, want string) string {
	if len(got)+len(want) < 40 {
		return fmt.Sprintf("got %s, want %s", got, want)
	}
	return fmt.Sprintf("\ngot  %s\nwant %s", got, want)
}
