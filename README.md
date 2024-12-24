# JB Logger

A high-performance, production-ready logging package for Go applications with automatic file rotation and structured logging capabilities.

## Features

- Automatic log file rotation based on size and line count
- Multiple log levels (DEBUG, INFO, WARN, ERROR, FATAL)
- Colored console output in development mode
- Asynchronous logging for high performance
- Stack trace support for error logging
- Configurable buffer size and rotation settings
- Thread-safe logging operations

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
)

func main() {
    err := logger.Initialize(logger.Config{
        LogPath:     "storage/logs/app",  // Will create app.1.log, app.2.log, etc.
        MaxFileSize: 100 * 1024 * 1024,   // 100MB
        Level:       logger.DEBUG,         // Minimum log level
        BufferSize:  500000,              // Channel buffer size
        IsDev:       false,               // Production mode
        MaxLines:    100000,              // Rotate every 100K lines
    })
    if err != nil {
        panic(err)
    }
    defer logger.Close()

    // Basic logging
    logger.Debug("Debug message: %v", someVar)
    logger.Info("Info message: %v", someVar)
    logger.Warn("Warning message: %v", someVar)
    logger.Error("Error message: %v", someVar)
    
    // Error with stack trace
    if err := someFunction(); err != nil {
        logger.ErrorWithStack("Operation failed", err)
    }
    
    // Fatal logging (will exit the program)
    logger.Fatal("Fatal error: %v", err)
}
```

## Configuration Options

- `LogPath`: Base path for log files (without extension)
- `MaxFileSize`: Maximum size of each log file before rotation (in bytes)
- `Level`: Minimum log level to record
- `BufferSize`: Size of the internal channel buffer for async logging
- `IsDev`: Development mode flag (enables console output)
- `MaxLines`: Maximum number of lines per file before rotation

## Log Levels

- DEBUG (Blue)
- INFO (Green)
- WARN (Yellow)
- ERROR (Red)
- FATAL (Purple)

## Log Rotation

Logs are automatically rotated when either of these conditions is met:
- File size exceeds MaxFileSize
- Line count exceeds MaxLines

Files are named with incrementing indices (e.g., app.1.log, app.2.log, etc.)

## Performance

The logger uses asynchronous writing and buffering for optimal performance. In benchmark tests, it can handle thousands of log entries per second with minimal impact on application performance.

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
