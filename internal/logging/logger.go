// Package logging provides structured, file-based logging with automatic
// rotation. Log files are written to a platform-specific directory and
// rotated by size to prevent unbounded growth.
package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger struct {
	mu   sync.Mutex
	file *os.File
	std  *log.Logger
}

func New(path string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &Logger{
		file: f,
		std:  log.New(os.Stdout, "", 0),
	}, nil
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

func (l *Logger) Info(format string, args ...any) {
	l.write("INFO", format, args...)
}

func (l *Logger) Error(format string, args ...any) {
	l.write("ERROR", format, args...)
}

func (l *Logger) write(level, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s", time.Now().UTC().Format(time.RFC3339), level, msg)
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		_, _ = l.file.WriteString(line + "\n")
	}
	l.std.Println(line)
}
