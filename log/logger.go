package log

import (
	"log"
	"os"
)

var logger *log.Logger

func Info(message string) {
	if logger == nil {
		logFile, err := os.OpenFile("editor.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			panic("error opening log file: " + err.Error())
		}
		logger = log.New(logFile, "", log.LstdFlags|log.Lshortfile)
	}
	logger.Println(message)
}
