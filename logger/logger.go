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

// Log levels
const (
	DEBUG = iota
	INFO
	WARN
	ERROR
	FATAL
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
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

type logEntry struct {
	level     int
	msg       []byte
	file      string
	line      int
	timestamp int64
}

type Logger struct {
	file         *os.File
	level        int
	logPath      string
	maxSize      int64
	currSize     int64
	logChan      chan *logEntry
	done         chan struct{}
	wg           sync.WaitGroup
	bufferSize   int
	isDev        bool // Development mode flag
	lineCount    int  // Current line count
	maxLines     int  // Maximum lines per file
	currentIndex int  // Current file index
}

type Config struct {
	LogPath     string // Path to log file
	MaxFileSize int64  // Maximum size of log file in bytes before rotation
	Level       int    // Minimum log level
	BufferSize  int    // Size of the log buffer channel
	IsDev       bool   // Development mode (enables console output)
	MaxLines    int    // Maximum lines per file before rotation (0 = no limit)
}

var defaultLogger *Logger

// Initialize creates a new logger instance
func Initialize(config Config) error {
	if config.LogPath == "" {
		pwd, _ := os.Getwd()
		config.LogPath = filepath.Join(pwd, "storage", "logs", "app")
	}

	// Ensure the base path doesn't have an extension
	ext := filepath.Ext(config.LogPath)
	if ext != "" {
		config.LogPath = strings.TrimSuffix(config.LogPath, ext)
	}

	if config.BufferSize == 0 {
		config.BufferSize = 100000
	}

	if config.MaxLines == 0 {
		config.MaxLines = 100000 // Default to 100K lines per file
	}

	// Create logs directory
	if err := os.MkdirAll(filepath.Dir(config.LogPath), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Get next available index
	nextIndex := 1
	baseDir := filepath.Dir(config.LogPath)
	baseName := filepath.Base(config.LogPath)

	files, err := os.ReadDir(baseDir)
	if err == nil {
		pattern := fmt.Sprintf("%s.*.log", baseName)
		for _, f := range files {
			if match, _ := filepath.Match(pattern, f.Name()); match {
				parts := strings.Split(strings.TrimSuffix(f.Name(), ".log"), ".")
				if len(parts) == 2 {
					if idx, err := strconv.Atoi(parts[1]); err == nil {
						if idx >= nextIndex {
							nextIndex = idx + 1
						}
					}
				}
			}
		}
	}

	// Open new log file with index
	logPath := fmt.Sprintf("%s.%d.log", config.LogPath, nextIndex)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	logger := &Logger{
		file:         file,
		level:        config.Level,
		logPath:      config.LogPath, // Store without extension
		maxSize:      config.MaxFileSize,
		currSize:     0,
		logChan:      make(chan *logEntry, config.BufferSize),
		done:         make(chan struct{}),
		wg:           sync.WaitGroup{},
		bufferSize:   config.BufferSize,
		isDev:        config.IsDev,
		maxLines:     config.MaxLines,
		lineCount:    0,
		currentIndex: nextIndex,
	}

	defaultLogger = logger
	logger.wg.Add(1)
	go logger.processLogs()

	return nil
}

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

		// Development mode: print to console with colors
		if l.isDev {
			fmt.Printf("%s [%s%s%s] [%s:%d] %s\n",
				time.Unix(0, entry.timestamp).Format("2006/01/02 15:04:05"),
				levelColors[entry.level],
				levelNames[entry.level],
				colorReset,
				relPath, entry.line,
				entry.msg)
		}

		// Always write to file with IDE-friendly path
		fmt.Fprintf(buf, "%s [%s] [%s:%d] %s\n",
			time.Unix(0, entry.timestamp).Format("2006/01/02 15:04:05"),
			levelNames[entry.level],
			relPath, entry.line,
			entry.msg)

		l.lineCount++
		if l.lineCount >= l.maxLines {
			l.rotate()
			l.lineCount = 0
		}
	}

	// Single write for entire batch
	if n, err := l.file.Write(buf.Bytes()); err != nil {
		if l.isDev {
			fmt.Printf("Error writing to log file: %v\n", err)
		}
	} else {
		l.currSize += int64(n)
		if l.currSize >= l.maxSize {
			l.rotate()
		}
	}
}

func (l *Logger) rotate() error {
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close current log file: %v", err)
	}

	// Increment index for next file
	l.currentIndex++

	// Create new log file with index
	newPath := fmt.Sprintf("%s.%d.log", l.logPath, l.currentIndex)

	file, err := os.OpenFile(newPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %v", err)
	}

	l.file = file
	l.currSize = 0
	l.lineCount = 0

	return nil
}

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
