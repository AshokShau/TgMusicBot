package ntgcalls

import (
	"fmt"
	"log"
	"os"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

type Logger struct {
	name  string
	level LogLevel
	l     *log.Logger
}

func NewLogger(name string, level LogLevel) *Logger {
	return &Logger{
		name:  name,
		level: level,
		l:     log.New(os.Stdout, fmt.Sprintf("[%s] ", name), log.LstdFlags),
	}
}

func (lg *Logger) log(level LogLevel, tag string, msg string) {
	if level >= lg.level {
		lg.l.Printf("[%s] %s", tag, msg)
	}
}

func (lg *Logger) Debug(msg string) { lg.log(LevelDebug, "DEBUG", msg) }
func (lg *Logger) Info(msg string)  { lg.log(LevelInfo, "INFO", msg) }
func (lg *Logger) Warn(msg string)  { lg.log(LevelWarn, "WARN", msg) }
func (lg *Logger) Error(msg string) { lg.log(LevelError, "ERROR", msg) }
func (lg *Logger) Fatal(msg string) { lg.log(LevelFatal, "FATAL", msg) }
