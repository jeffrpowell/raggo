package logging

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Logger struct {
	prefix string
	logger *log.Logger
}

func New(prefix string) *Logger {
	return &Logger{
		prefix: prefix,
		logger: log.New(os.Stderr, "", 0),
	}
}

func (l *Logger) log(level, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[%s] [%s] [%s] %s", timestamp, level, l.prefix, msg)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log("INFO", format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log("WARN", format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log("ERROR", format, args...)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log("FATAL", format, args...)
	os.Exit(1)
}
