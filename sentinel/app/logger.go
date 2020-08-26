package app

import (
	"log"
	"os"
)

type Logger interface {
	Println(...interface{})
	Printf(string, ...interface{})
	Fatal(...interface{})
	Fatalf(string, ...interface{})
}

func NewDefaultLogger() Logger {
	return log.New(os.Stderr, "", log.LstdFlags)
}
