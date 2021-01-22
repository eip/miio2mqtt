package helpers

import (
	"encoding/hex"
	"time"

	log "github.com/go-pkgz/lgr"
)

type TestLog struct {
	Message string
}

func (m *TestLog) Write(p []byte) (n int, err error) {
	m.Message = string(p)
	return len(p), nil
}

func (m *TestLog) Reset() {
	m.Message = ""
}

func InitTestLog() *TestLog {
	logger := &TestLog{}
	log.Setup(log.Format(`[{{.Level}}] {{.Message}}`), log.Out(logger))
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
