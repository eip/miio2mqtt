package main

import (
	"fmt"

	prompt "github.com/c-bata/go-prompt"
	log "github.com/go-pkgz/lgr"
)

const logPrefix = "    "
const logColor = prompt.DarkGray

var stdOut = prompt.NewStdoutWriter()

func colorPrintf(fg prompt.Color, format string, a ...interface{}) {
	stdOut.SetColor(fg, prompt.DefaultColor, false)
	stdOut.WriteStr(fmt.Sprintf(format, a...))
	stdOut.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	stdOut.Flush()
}

type colorLog struct{}

func (l *colorLog) Write(p []byte) (n int, err error) {
	stdOut.SetColor(logColor, prompt.DefaultColor, false)
	stdOut.WriteStr(logPrefix)
	stdOut.Write(p)
	stdOut.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	stdOut.Flush()
	return len(logPrefix) + len(p), nil
}

type nilLog struct{}

func (l *nilLog) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func setupLog() {
	stripDate := log.Mapper{TimeFunc: func(s string) string { return s[11:] }}
	log.Setup(log.Debug, log.Msec, log.LevelBraces, log.Map(stripDate), log.Out(&colorLog{}), log.Err(&nilLog{}))
}
