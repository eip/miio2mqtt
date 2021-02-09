package helpers

import (
	"encoding/hex"
	"regexp"

	log "github.com/go-pkgz/lgr"
)

type TestLog struct {
	Message string
}

func (l *TestLog) Write(p []byte) (n int, err error) {
	l.Message = string(p)
	return len(p), nil
}

func (l *TestLog) Reset() {
	l.Message = ""
}

type nilLog struct{}

func (l *nilLog) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func InitTestLog() *TestLog {
	logger := &TestLog{}
	log.Setup(log.Format(`[{{.Level}}] {{.Message}}`), log.Out(logger), log.Err(&nilLog{}))
	return logger
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

var reIsJSON = regexp.MustCompile(`(?i)^{\s*"[a-z_]+":.+}$`)
var reJSONKey = regexp.MustCompile(`(?i)"([a-z_]+)":`)

func IsJSON(data []byte) bool {
	return reIsJSON.Match(data)
}

func StripJSONQuotes(data []byte) []byte {
	return reJSONKey.ReplaceAll(data, []byte("$1:"))
}
