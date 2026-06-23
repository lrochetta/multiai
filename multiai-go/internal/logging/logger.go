package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
}

type Logger struct {
	mu       sync.Mutex
	logDir   string
	minLevel Level
}

var defaultLogger *Logger

func init() {
	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, ".multiai", "logs")
	os.MkdirAll(logDir, 0700)
	defaultLogger = &Logger{logDir: logDir, minLevel: INFO}
}

func Get() *Logger { return defaultLogger }

func (l *Logger) Log(level Level, format string, args ...interface{}) {
	if level < l.minLevel {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), levelNames[level], msg)

	// Write to file
	logFile := filepath.Join(l.logDir, "multiai.log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(line)

	// Also print to stderr for WARN and ERROR
	if level >= WARN {
		os.Stderr.WriteString(line)
	}
}

func Debug(format string, args ...interface{}) { Get().Log(DEBUG, format, args...) }
func Info(format string, args ...interface{})  { Get().Log(INFO, format, args...) }
func Warn(format string, args ...interface{})  { Get().Log(WARN, format, args...) }
func Error(format string, args ...interface{}) { Get().Log(ERROR, format, args...) }
