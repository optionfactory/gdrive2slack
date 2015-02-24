package gdrive2slack

import (
	"io"
	"log"
)

type Logger struct {
	real *log.Logger
}

func NewLogger(out io.Writer, prefix string, flag int) *Logger {
	return &Logger{
		log.New(out, prefix, flag),
	}
}

func (self *Logger) Info(format string, v ...interface{}) {
	self.real.Printf("[INFO ] "+format+"\n", v...)
}

func (self *Logger) Warning(format string, v ...interface{}) {
	self.real.Printf("[WARN ] "+format+"\n", v...)
}

func (self *Logger) Error(format string, v ...interface{}) {
	self.real.Printf("[ERROR] "+format+"\n", v...)
}
