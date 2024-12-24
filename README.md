# JB Logger

A high-performance, production-ready logging package for Go applications with automatic file rotation and structured logging capabilities. Built with focus on performance, reliability, and ease of use, this logger provides both file-based logging with automatic rotation and colored console output for development.

## Key Features

- **Automatic Log Rotation**: Based on file size or line count
- **Colored Console Output**: Different colors for each log level
- **Asynchronous Logging**: High-performance non-blocking operations
- **Buffered Channels**: Configurable buffer size for optimal performance
- **Stack Traces**: Detailed stack traces for error debugging
- **Thread-Safe**: Safe for concurrent use
- **Structured Format**: Consistent, easy-to-parse log format
- **Configurable**: Flexible configuration options

## Installation

To install the latest version:
```bash
go get github.com/jbarasa/logger@latest
```

To install a specific version:
```bash
go get github.com/jbarasa/logger@v1.0.0
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
    "time"
)

func main() {
    // Initialize logger with configuration
    err := logger.Initialize(logger.Config{
        LogPath:     "storage/logs/app",  // Will create app.1.log, app.2.log, etc.
        MaxFileSize: 100 * 1024 * 1024,   // 100MB
        Level:       logger.DEBUG,         // Minimum log level
        BufferSize:  500000,              // Channel buffer size
        IsDev:       true,                // Enable console output with colors
        MaxLines:    100000,              // Rotate every 100K lines
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

## Example Output

### Console Output (Development Mode)
```
[2024/12/25 02:45:40] [DEBUG] [main.go:25] Debug message: Starting application
[2024/12/25 02:45:40] [INFO]  [main.go:26] Server started on port 8080
[2024/12/25 02:45:40] [WARN]  [main.go:27] High memory usage: 85%
[2024/12/25 02:45:40] [ERROR] [main.go:28] Failed to connect to database: connection timeout
[2024/12/25 02:45:40] [ERROR] [main.go:31] Database operation failed
Stack Trace:
goroutine 1 [running]:
main.main()
    /path/to/main.go:31
...
```

### Log File (app.1.log)
```
2024/12/25 02:45:40 [DEBUG] [main.go:25] Debug message: Starting application
2024/12/25 02:45:40 [INFO]  [main.go:26] Server started on port 8080
2024/12/25 02:45:40 [WARN]  [main.go:27] High memory usage: 85%
2024/12/25 02:45:40 [ERROR] [main.go:28] Failed to connect to database: connection timeout
2024/12/25 02:45:40 [ERROR] [main.go:31] Database operation failed
Stack Trace:
goroutine 1 [running]:
main.main()
    /path/to/main.go:31
...
```

### Log Colors (Console)
- DEBUG: Blue
- INFO: Green
- WARN: Yellow
- ERROR: Red
- FATAL: Purple

## Configuration Options

- `LogPath`: Base path for log files (without extension)
  - Example: "storage/logs/app" creates app.1.log, app.2.log, etc.
  - Files rotate automatically based on size or line count

- `MaxFileSize`: Maximum size of each log file before rotation (in bytes)
  - Default: 100MB
  - Example: `100 * 1024 * 1024` for 100MB

- `Level`: Minimum log level to record
  - Available levels: DEBUG, INFO, WARN, ERROR, FATAL
  - Messages below this level are ignored

- `BufferSize`: Size of the internal channel buffer for async logging
  - Default: 100000
  - Larger values can improve performance but use more memory

- `IsDev`: Development mode flag
  - When true: Enables colored console output
  - When false: Logs only to files

- `MaxLines`: Maximum number of lines per file before rotation
  - Default: 100000
  - Set to 0 to disable line-based rotation

## Log Rotation

Logs are automatically rotated when either of these conditions is met:
- File size exceeds MaxFileSize
- Line count exceeds MaxLines

Files are named with incrementing indices:
```
app.1.log  (current)
app.2.log  (previous)
app.3.log  (older)
...
```

## Performance

The logger uses asynchronous writing and buffering for optimal performance:
- Non-blocking log calls (return immediately)
- Buffered channel for log entries
- Batch writing to improve I/O performance
- Automatic file rotation without blocking

Benchmark results on a typical system:
- Over 100,000 log entries per second
- Minimal impact on application performance
- Memory efficient with configurable buffer sizes

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

GNU General Public License v3.0 - see LICENSE file for details

## Version History

- v1.0.0: Initial release
  - Automatic log rotation
  - Multiple log levels
  - Async logging
  - Stack trace support

## Versioning

This project follows [Semantic Versioning](https://semver.org/). Version numbers are in the format MAJOR.MINOR.PATCH:
- MAJOR version for incompatible API changes
- MINOR version for added functionality in a backwards compatible manner
- PATCH version for backwards compatible bug fixes

To update to a new version in your project:
1. Run `go get -u github.com/jbarasa/logger@latest` for the latest version
2. Or specify a version: `go get github.com/jbarasa/logger@v1.0.2`
