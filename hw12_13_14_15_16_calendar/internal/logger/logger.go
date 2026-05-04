package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Logger — обёртка над slog.
type Logger struct {
	*slog.Logger
}

// New создаёт логгер с указанным уровнем.
func New(level string) *Logger {
	return NewWithOutput(level, os.Stdout)
}

// NewWithOutput позволяет указать io.Writer (для тестов).
func NewWithOutput(level string, out io.Writer) *Logger {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "info", "":
		l = slog.LevelInfo
	case "warn", "warning":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: l}
	handler := slog.NewTextHandler(out, opts)

	return &Logger{
		Logger: slog.New(handler).With("service", "calendar"),
	}
}

// Helper-методы.
func (l *Logger) Info(msg string) {
	l.Logger.Info(msg)
}

func (l *Logger) Error(msg string) {
	l.Logger.Error(msg)
}

func (l *Logger) Debug(msg string) {
	l.Logger.Debug(msg)
}

func (l *Logger) Warn(msg string) {
	l.Logger.Warn(msg)
}

func (l *Logger) With(key, value string) *Logger {
	return &Logger{
		Logger: l.Logger.With(key, value),
	}
}
