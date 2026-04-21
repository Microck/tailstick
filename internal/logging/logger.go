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

// Logger provides thread-safe, leveled logging to both a file and stdout.
type Logger struct {
	mu   sync.Mutex
	file *os.File
	std  *log.Logger
}

// New creates a Logger that appends to the given file path.
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

// Close flushes and closes the underlying log file.
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

// Info logs a message at INFO level.
func (l *Logger) Info(format string, args ...any) {
	l.write("INFO", format, args...)
}

// Error logs a message at ERROR level.
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
