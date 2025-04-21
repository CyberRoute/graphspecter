// Package logger provides structured logging for the GraphSpecter tool
package logger

import (
	"fmt"
	"io"
	"os"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel int

// Log levels
const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

var (
	// currentLevel is the current log level
	currentLevel = LevelInfo

	// output is where log messages are written
	output io.Writer = os.Stdout

	// logFile is the file handler for log file output
	logFile *os.File

	// logLevelStrings maps log levels to their string representations
	logLevelStrings = map[LogLevel]string{
		LevelDebug: "DEBUG",
		LevelInfo:  "INFO",
		LevelWarn:  "WARN",
		LevelError: "ERROR",
		LevelFatal: "FATAL",
	}

	// logLevelColors maps log levels to ANSI color codes for terminal output
	logLevelColors = map[LogLevel]string{
		LevelDebug: "\033[37m", // White
		LevelInfo:  "\033[32m", // Green
		LevelWarn:  "\033[33m", // Yellow
		LevelError: "\033[31m", // Red
		LevelFatal: "\033[35m", // Magenta
	}

	// colorReset is the ANSI code to reset color
	colorReset = "\033[0m"

	// useColors determines if color should be used in log output
	useColors = true
)

// SetLevel sets the current log level
func SetLevel(level LogLevel) {
	currentLevel = level
}

// SetOutput sets the output writer for logs
func SetOutput(w io.Writer) {
	output = w
}

// SetupLogging configures logging from CLI flags.
// level: "debug", "info", "warn", "error", "fatal" (only that level appears)
// logFilePath: path for file output (append)
// enableColors: whether to colorize terminal output
func SetupLogging(level string, logFilePath string, enableColors bool) {
	switch level {
	case "debug":
		SetLevel(LevelDebug)
	case "info":
		SetLevel(LevelInfo)
	case "warn":
		SetLevel(LevelWarn)
	case "error":
		SetLevel(LevelError)
	case "fatal":
		SetLevel(LevelFatal)
	default:
		SetLevel(LevelInfo)
	}

	if logFilePath != "" {
		if err := SetLogFile(logFilePath); err != nil {
			fmt.Printf("Error setting up log file: %v\n", err)
			os.Exit(1)
		}
	}
	// Enable or disable color output.
	EnableColors(enableColors)
}

// SetLogFile sets up logging to a file in addition to stdout
func SetLogFile(filename string) error {
	if logFile != nil {
		logFile.Close()
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	logFile = file
	output = io.MultiWriter(os.Stdout, logFile)
	return nil
}

// CloseLogFile closes the log file if one is open
func CloseLogFile() {
	if logFile != nil {
		logFile.Close()
		logFile = nil
		output = os.Stdout
	}
}

// EnableColors toggles colored output
func EnableColors(enable bool) {
	useColors = enable
}

// log formats and writes a log message at the given level.
// Only messages exactly matching currentLevel are emitted.
func log(level LogLevel, format string, args ...interface{}) {
	// Suppress any log that does not match the current log level.
	if level != currentLevel {
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	levelStr := logLevelStrings[level]
	var entry string

	if useColors && output == os.Stdout {
		color := logLevelColors[level]
		entry = fmt.Sprintf("%s [%s%s%s] %s\n", now, color, levelStr, colorReset, msg)
	} else {
		entry = fmt.Sprintf("%s [%s] %s\n", now, levelStr, msg)
	}

	fmt.Fprint(output, entry)

	if level == LevelFatal {
		if logFile != nil {
			logFile.Close()
		}
		os.Exit(1)
	}
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	log(LevelDebug, format, args...)
}

// Info logs an informational message
func Info(format string, args ...interface{}) {
	log(LevelInfo, format, args...)
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	log(LevelWarn, format, args...)
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	log(LevelError, format, args...)
}

// Fatal logs a fatal error message and exits the program
func Fatal(format string, args ...interface{}) {
	log(LevelFatal, format, args...)
}
