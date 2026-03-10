/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ntgcalls

import (
	"log/slog"
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
	l     *slog.Logger
}

func NewLogger(name string, level LogLevel) *Logger {
	return &Logger{
		name:  name,
		level: level,
		l:     slog.Default().With("logger", name),
	}
}

func (lg *Logger) log(level LogLevel, msg string) {
	if level < lg.level {
		return
	}
	switch level {
	case LevelDebug:
		lg.l.Debug(msg)
	case LevelInfo:
		lg.l.Info(msg)
	case LevelWarn:
		lg.l.Warn(msg)
	case LevelError:
		lg.l.Error(msg)
	case LevelFatal:
		lg.l.Error(msg)
	}
}

func (lg *Logger) Debug(msg string) { lg.log(LevelDebug, msg) }
func (lg *Logger) Info(msg string)  { lg.log(LevelInfo, msg) }
func (lg *Logger) Warn(msg string)  { lg.log(LevelWarn, msg) }
func (lg *Logger) Error(msg string) { lg.log(LevelError, msg) }
func (lg *Logger) Fatal(msg string) { lg.log(LevelFatal, msg) }
