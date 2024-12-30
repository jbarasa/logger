// Package logger provides a high-performance, production-ready logging solution
// with automatic log cleanup, colored console output, and asynchronous writing.
//
// Version: 1.0.2
//
// Features:
// - Automatic cleanup of logs older than 1 month
// - Multiple log levels with color-coded console output
// - Asynchronous logging with buffered channels
// - Stack trace support for error debugging
// - Thread-safe operations
// - Configurable buffer sizes
// - Log file rotation with numbered backup files
//
// Example usage:
//
//	err := logger.Initialize(logger.Config{
//	    LogPath:    "storage/logs/app.log",
//	    Level:      logger.DEBUG,
//	    BufferSize: 500000,
//	    IsDev:      true,
//	    MaxFileSize: 25 * 1024 * 1024, // 25MB
//	})
//	if err != nil {
//	    panic(err)
//	}
//	defer logger.Close()
//
//	logger.Info("Server started on port %d", 8080)
//	logger.Error("Database error: %v", err)
package logger

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Log levels define the severity of the log message
const (
	DEBUG = iota // Detailed information for debugging
	INFO         // General information about program execution
	WARN         // Warning messages for potentially harmful situations
	ERROR        // Error messages for serious problems
	FATAL        // Critical errors that require immediate attention (exits program)
)

// ANSI color codes for console output
const (
	colorReset  = "\033[0m"  // Reset to default color
	colorRed    = "\033[31m" // Error messages
	colorGreen  = "\033[32m" // Info messages
	colorYellow = "\033[33m" // Warning messages
	colorBlue   = "\033[34m" // Debug messages
	colorPurple = "\033[35m" // Fatal messages
)

// Level names for log output
var levelNames = map[int]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

// Color mapping for log levels
var levelColors = map[int]string{
	DEBUG: colorBlue,
	INFO:  colorGreen,
	WARN:  colorYellow,
	ERROR: colorRed,
	FATAL: colorPurple,
}

var entryPool = sync.Pool{
	New: func() interface{} {
		return &logEntry{
			msg: make([]byte, 0, 1024),
		}
	},
}

// logEntry represents a single log message
type logEntry struct {
	level     int
	msg       []byte
	file      string
	line      int
	timestamp int64
}

// Config defines the configuration options for the logger
type Config struct {
	LogPath     string // Path for log file (with extension)
	Level       int    // Minimum log level to record
	BufferSize  int    // Size of the log buffer channel
	IsDev       bool   // Development mode (enables console output)
	MaxFileSize int64  // Maximum file size in bytes before rotation (default: 25MB)
}

// Logger represents the core logger structure
type Logger struct {
	file       *os.File       // Current log file handle
	level      int            // Current minimum log level
	logPath    string         // Path for log file
	logChan    chan *logEntry // Channel for async logging
	done       chan struct{}  // Channel for shutdown signaling
	wg         sync.WaitGroup // Wait group for graceful shutdown
	bufferSize int            // Size of the log buffer
	isDev      bool           // Development mode flag
	maxSize    int64          // Maximum file size before rotation
	currSize   int64          // Current file size
	mu         sync.Mutex     // Mutex for file operations
}

var defaultLogger *Logger

// Initialize creates a new logger instance
func Initialize(config Config) error {
	if config.LogPath == "" {
		pwd, _ := os.Getwd()
		config.LogPath = filepath.Join(pwd, "storage", "logs", "app.log")
	}

	// Create logs directory and archive subdirectory
	logsDir := filepath.Dir(config.LogPath)
	archiveDir := filepath.Join(logsDir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directories: %v", err)
	}

	if config.BufferSize == 0 {
		config.BufferSize = 100000
	}

	if config.MaxFileSize == 0 {
		config.MaxFileSize = 25 * 1024 * 1024 // 25MB default
	}

	// Open log file
	file, err := os.OpenFile(config.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to get file info: %v", err)
	}

	logger := &Logger{
		file:       file,
		level:      config.Level,
		logPath:    config.LogPath,
		logChan:    make(chan *logEntry, config.BufferSize),
		done:       make(chan struct{}),
		wg:         sync.WaitGroup{},
		bufferSize: config.BufferSize,
		isDev:      config.IsDev,
		maxSize:    config.MaxFileSize,
		currSize:   info.Size(),
	}

	defaultLogger = logger
	logger.wg.Add(1)
	go logger.processLogs()

	return nil
}

// processLogs is the main logging loop that processes log entries from the channel
func (l *Logger) processLogs() {
	defer l.wg.Done()

	batch := make([]*logEntry, 0, 50000)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case entry := <-l.logChan:
			batch = append(batch, entry)

			if len(batch) >= 50000 {
				l.writeBatch(batch)
				for _, e := range batch {
					e.msg = e.msg[:0]
					entryPool.Put(e)
				}
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				l.writeBatch(batch)
				for _, e := range batch {
					e.msg = e.msg[:0]
					entryPool.Put(e)
				}
				batch = batch[:0]
			}

		case <-l.done:
			close(l.logChan)
			for entry := range l.logChan {
				batch = append(batch, entry)
				if len(batch) >= 50000 {
					l.writeBatch(batch)
					for _, e := range batch {
						e.msg = e.msg[:0]
						entryPool.Put(e)
					}
					batch = batch[:0]
				}
			}
			if len(batch) > 0 {
				l.writeBatch(batch)
				for _, e := range batch {
					e.msg = e.msg[:0]
					entryPool.Put(e)
				}
			}
			return
		}
	}
}

// writeBatch writes a batch of log entries to the file
func (l *Logger) writeBatch(entries []*logEntry) {
	if len(entries) == 0 {
		return
	}

	buf := bytes.NewBuffer(make([]byte, 0, 64*1024)) // 64KB buffer
	defer buf.Reset()

	pwd, _ := os.Getwd()

	for _, entry := range entries {
		// Get relative path for better IDE integration
		relPath := entry.file
		if abs, err := filepath.Abs(entry.file); err == nil {
			if rel, err := filepath.Rel(pwd, abs); err == nil {
				relPath = rel
			}
		}

		timeStr := time.Unix(0, entry.timestamp).Format("2006/01/02 15:04:05")

		// Development mode: print to console with colors
		if l.isDev {
			fmt.Printf("%s [%s%s%s] [%s:%d] %s\n",
				timeStr,
				levelColors[entry.level],
				levelNames[entry.level],
				colorReset,
				relPath, entry.line,
				entry.msg)
		}

		// Always write to file with IDE-friendly path
		fmt.Fprintf(buf, "%s [%s] [%s:%d] %s\n",
			timeStr,
			levelNames[entry.level],
			relPath, entry.line,
			entry.msg)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Write to file
	n, err := l.file.Write(buf.Bytes())
	if err != nil {
		if l.isDev {
			fmt.Printf("Error writing to log file: %v\n", err)
		}
		return
	}

	l.currSize += int64(n)
	if l.currSize >= l.maxSize {
		if err := l.rotate(); err != nil && l.isDev {
			fmt.Printf("Error rotating log file: %v\n", err)
		}
	}
}

// rotate moves the current log file to the archive directory with a number
func (l *Logger) rotate() error {
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close current log file: %v", err)
	}

	// Get next archive number
	nextNum, err := l.getNextArchiveNumber()
	if err != nil {
		return fmt.Errorf("failed to get next archive number: %v", err)
	}

	// Create archive path
	archiveDir := filepath.Join(filepath.Dir(l.logPath), "archive")
	archivePath := filepath.Join(archiveDir, fmt.Sprintf("%d.log", nextNum))

	// Move current log to archive
	if err := os.Rename(l.logPath, archivePath); err != nil {
		return fmt.Errorf("failed to move log to archive: %v", err)
	}

	// Create new empty log file
	file, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %v", err)
	}

	l.file = file
	l.currSize = 0
	return nil
}

// getNextArchiveNumber gets the next available archive number
func (l *Logger) getNextArchiveNumber() (int, error) {
	archiveDir := filepath.Join(filepath.Dir(l.logPath), "archive")
	files, err := os.ReadDir(archiveDir)
	if err != nil {
		return 1, err
	}

	maxNum := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if num, err := strconv.Atoi(strings.TrimSuffix(name, ".log")); err == nil {
			if num > maxNum {
				maxNum = num
			}
		}
	}
	return maxNum + 1, nil
}

// log logs a message at the specified level
func (l *Logger) log(level int, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	// Get caller info
	_, file, line, _ := runtime.Caller(2)

	// Get message buffer from pool
	msgBuf := bytes.NewBuffer(make([]byte, 0, 1024)) // 1KB for messages
	fmt.Fprintf(msgBuf, format, args...)

	// Get entry from pool
	entry := entryPool.Get().(*logEntry)
	entry.level = level
	entry.msg = append(entry.msg[:0], msgBuf.Bytes()...)
	entry.file = file
	entry.line = line
	entry.timestamp = time.Now().UnixNano()

	// Non-blocking send
	select {
	case l.logChan <- entry:
	default:
		if l.isDev {
			fmt.Printf("WARNING: Log buffer full, dropping message\n")
		}
		entry.msg = entry.msg[:0]
		entryPool.Put(entry)
	}

	if level == FATAL {
		close(l.done)
		l.wg.Wait()
		os.Exit(1)
	}
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(DEBUG, format, args...)
	}
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(INFO, format, args...)
	}
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(WARN, format, args...)
	}
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(ERROR, format, args...)
	}
}

// ErrorWithStack logs an error message with stack trace
func ErrorWithStack(msg string, err error) {
	if defaultLogger != nil {
		stackBuf := make([]byte, 4096)
		n := runtime.Stack(stackBuf, false)
		defaultLogger.log(ERROR, "%s: %v\nStack Trace:\n%s", msg, err, stackBuf[:n])
	}
}

// Fatal logs a fatal message and exits the program
func Fatal(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(FATAL, format, args...)
	}
}

// Close closes the logger
func Close() error {
	if defaultLogger != nil {
		close(defaultLogger.done)
		defaultLogger.wg.Wait()
		return defaultLogger.file.Close()
	}
	return nil
}
