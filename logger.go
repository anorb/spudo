package spudo

import (
	"fmt"
	"log"
	"os"
)

type spudoLogger struct {
	infologger  *log.Logger
	errorlogger *log.Logger
}

func newLogger() (l *spudoLogger) {
	l = new(spudoLogger)
	l.infologger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	l.errorlogger = log.New(os.Stdout, "ERR: ", log.Ldate|log.Ltime|log.Lshortfile)
	return
}

func (l spudoLogger) info(msg string, extra ...interface{}) {
	l.infologger.Print(msg, fmt.Sprintln(extra...))
}

func (l spudoLogger) error(msg string, extra ...interface{}) {
	l.errorlogger.Print(msg, fmt.Sprintln(extra...))
}

func (l spudoLogger) fatal(msg string, extra ...interface{}) {
	l.errorlogger.Fatal(msg, fmt.Sprintln(extra...))
}
