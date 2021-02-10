package helpers

import (
	"bytes"
	"encoding/hex"
	"regexp"

	log "github.com/go-pkgz/lgr"
	"github.com/sergi/go-diff/diffmatchpatch"
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

func IsJSON(data string) bool {
	return reIsJSON.MatchString(data)
}

func StripJSONQuotes(data string) string {
	return reJSONKey.ReplaceAllString(data, "$1:")
}

func DiffStrings(old, new, color string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(old, new, false)
	return diffPretty(diffs, color)
}

func diffPretty(diffs []diffmatchpatch.Diff, color string) string {
	var buff bytes.Buffer
	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			_, _ = buff.WriteString("\x1b[" + color + "m" + diff.Text + "\x1b[0m")
		case diffmatchpatch.DiffEqual:
			_, _ = buff.WriteString(diff.Text)
		}
	}
	return buff.String()
}
