package log

import (
	"io"
	"log"
	"os"
)

var DEFAULT_OUTPUT io.Writer = os.Stdout

func SetDefaultOutput(io io.Writer) {
	DEFAULT_OUTPUT = io
}

func CreateLogger(componentName string) *log.Logger {
	return log.New(DEFAULT_OUTPUT, "["+componentName+"] ", log.LstdFlags|log.Lshortfile)
}
