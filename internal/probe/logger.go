package probe

import (
	"fmt"
)

// Logger structure with configuration options
type Logger struct {
	Verbose bool
	Quiet   bool
}

// LogLevel represents different logging levels
type LogLevel string

const (
	LevelDebug   LogLevel = "DEBUG"
	LevelInfo    LogLevel = "*"
	LevelSuccess LogLevel = "+"
	LevelWarning LogLevel = "-"
	LevelError   LogLevel = "!"
)

// log is the internal method that handles all logging
func (l *Logger) log(level LogLevel, format string, args ...any) {
	// Skip logging debug messages if not in verbose mode
	if level == LevelDebug && !l.Verbose {
		return
	}

	// Skip other messages if in quiet mode
	if level != LevelDebug && l.Quiet {
		return
	}

	// Format the message with any arguments
	message := fmt.Sprintf(format, args...)

	// Print the formatted message with appropriate prefix
	fmt.Printf("[%s] %s\n", level, message)
}

// Debug logs debug information (when verbose is true)
func (l *Logger) Debug(format string, args ...any) {
	l.log(LevelDebug, format, args...)
}

// Info logs general information (unless quiet is true)
func (l *Logger) Info(format string, args ...any) {
	l.log(LevelInfo, format, args...)
}

// Success logs success information (unless quiet is true)
func (l *Logger) Success(format string, args ...any) {
	l.log(LevelSuccess, format, args...)
}

// Warning logs warning information (unless quiet is true)
func (l *Logger) Warning(format string, args ...any) {
	l.log(LevelWarning, format, args...)
}

// Error logs error information (unless quiet is true)
func (l *Logger) Error(format string, args ...any) {
	l.log(LevelError, format, args...)
}
