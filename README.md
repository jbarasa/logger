# JB Logger

A high-performance, production-ready logging package for Go applications with automatic file rotation and structured logging capabilities. Built with focus on performance, reliability, and ease of use, this logger provides both file-based logging with automatic rotation and colored console output for development.

## Key Features

- **Automatic Log Rotation**: Rotates logs when file size reaches 25MB (configurable)
- **Organized Archive**: Rotated logs are stored in numbered files (1.log, 2.log, etc.)
- **Colored Console Output**: Different colors for each log level
- **Asynchronous Logging**: High-performance non-blocking operations
- **Buffered Channels**: Configurable buffer size for optimal performance
- **Stack Traces**: Detailed stack traces for error debugging
- **Thread-Safe**: Safe for concurrent use
- **Structured Format**: Consistent, easy-to-parse log format

## Installation

To install the latest version:
```bash
go get github.com/jbarasa/logger@latest
```

To install a specific version:
```bash
go get github.com/jbarasa/logger@v1.0.2
```

To update to the latest version in an existing project:
```bash
go get -u github.com/jbarasa/logger@latest
```

## Usage

```go
package main

import (
    "github.com/jbarasa/logger/logger"
    "errors"
)

func main() {
    // Initialize logger with configuration
    err := logger.Initialize(logger.Config{
        LogPath:     "storage/logs/app.log",
        MaxFileSize: 25 * 1024 * 1024,    // 25MB
        Level:       logger.DEBUG,         // Minimum log level
        BufferSize:  500000,              // Channel buffer size
        IsDev:       true,                // Enable console output with colors
    })
    if err != nil {
        panic(err)
    }
    defer logger.Close()

    // Example logging
    logger.Debug("Debug message: Starting application")
    logger.Info("Server started on port %d", 8080)
    logger.Warn("High memory usage: %d%%", 85)
    logger.Error("Failed to connect to database: %v", errors.New("connection timeout"))
    
    // Error with stack trace
    err = errors.New("critical database error")
    logger.ErrorWithStack("Database operation failed", err)
}
```

## Log Format

### Console Output (Development Mode)
```
2024/12/30 22:45:40 [DEBUG] [main.go:25] Debug message: Starting application
2024/12/30 22:45:40 [INFO]  [main.go:26] Server started on port 8080
2024/12/30 22:45:40 [WARN]  [main.go:27] High memory usage: 85%
2024/12/30 22:45:40 [ERROR] [main.go:28] Failed to connect to database: connection timeout
```

### Log Colors (Console)
- DEBUG: Blue
- INFO: Green
- WARN: Yellow
- ERROR: Red
- FATAL: Purple

## Configuration Options

- `LogPath`: Path for the log file (with extension)
  - Example: "storage/logs/app.log"
  - When rotated, old logs move to "storage/logs/archive/N.log"

- `MaxFileSize`: Maximum size of log file before rotation (in bytes)
  - Default: 25MB (25 * 1024 * 1024 bytes)
  - When reached, current log is moved to archive and new file is created

- `Level`: Minimum log level to record
  - Available levels: DEBUG, INFO, WARN, ERROR, FATAL
  - Messages below this level are ignored

- `BufferSize`: Size of the internal channel buffer for async logging
  - Default: 100000
  - Larger values can improve performance but use more memory

- `IsDev`: Development mode flag
  - When true: Enables colored console output
  - When false: Logs only to files

## Log Rotation

Logs are automatically rotated when file size exceeds MaxFileSize. The rotation process:
1. Current log file (app.log) reaches size limit
2. File is moved to archive/1.log (or next available number)
3. New empty app.log is created
4. Logging continues to new file

Archive structure:
```
storage/
  └── logs/
      ├── app.log     (current log file)
      └── archive/
          ├── 1.log   (oldest)
          ├── 2.log
          └── 3.log   (newest)
```

## Performance

The logger uses several techniques for optimal performance:
- Non-blocking log calls using buffered channels
- Batch writing to improve I/O performance
- Efficient file rotation with minimal locking
- Memory-efficient buffer management

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

GNU General Public License v3.0 - see LICENSE file for details

## Version History

- v1.0.2: (2024-12-30)
  - Simplified log rotation with archive directory
  - Fixed memory usage in file operations
  - Improved thread safety
  - Removed cleanup functionality in favor of simple rotation

- v1.0.1: Bug fixes and performance improvements

- v1.0.0: Initial release
  - Basic logging functionality
  - Multiple log levels
  - Async logging
  - Stack trace support
