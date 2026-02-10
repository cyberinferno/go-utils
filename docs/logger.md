# Logger Documentation

The `logger` package provides a structured logging interface backed by [zerolog](https://github.com/rs/zerolog). It supports console-only logging or console plus daily-rotated log files, with consistent levels (Debug, Info, Warn, Error) and key-value fields for context.

## Features

- **Structured Logging**: Attach key-value fields to every log entry
- **Log Levels**: Debug, Info, Warn, Error with configurable minimum level
- **zerolog Backend**: Fast, zero-allocation JSON (or console) output
- **Daily File Rotation**: Optional file output with automatic rotation by date
- **Request-Scoped Loggers**: Derive child loggers with `With()` for request IDs or component names
- **Service Tagging**: Add a service name to all entries for multi-service environments
- **Resource Cleanup**: `Close()` releases file handles; safe to call multiple times

## Installation

```go
import "github.com/cyberinferno/go-utils/logger"
```

Ensure the zerolog dependency is in your module:

```bash
go get github.com/cyberinferno/go-utils
```

## Creating a Logger

### Console-Only Logger (NewZerologLogger)

Use this when you already have a zerolog.Logger (e.g. writing to stdout) and want to wrap it with a service name, timestamp, and level filter. No files are created.

```go
import (
    "os"
    "github.com/cyberinferno/go-utils/logger"
    "github.com/rs/zerolog"
)

// Create zerolog logger writing to stdout
zlog := zerolog.New(os.Stdout)

// Wrap with service name and level
log := logger.NewZerologLogger(zlog, "my-service", zerolog.InfoLevel)

log.Info("server started")
// Output includes "service":"my-service", timestamp, and message
```

**Parameters:**

- **l**: The zerolog.Logger to wrap (e.g. from `zerolog.New(os.Stdout)`)
- **serviceName**: Name of the service; added as a field to every log entry
- **level**: Minimum level to log (e.g. `zerolog.InfoLevel`, `zerolog.DebugLevel`)

**Returns:**

- A `Logger` that writes through the given zerolog instance

### File + Console Logger (NewZerologFileLogger)

Use this when you want logs in both stdout and daily-rotated files. Log files are named `{serviceName}_{date}.log` (e.g. `my-service_2026-02-10.log`). The directory is created if it does not exist.

```go
import (
    "github.com/cyberinferno/go-utils/logger"
    "github.com/rs/zerolog"
)

log := logger.NewZerologFileLogger(
    "my-service",   // service name
    "/var/log/app", // log directory (created if missing)
    zerolog.InfoLevel,
)

log.Info("server started")
// Writes to stdout and to /var/log/app/my-service_2026-02-10.log
```

**Parameters:**

- **serviceName**: Name of the service; used in log entries and file names
- **logDir**: Directory for log files; created if it does not exist
- **level**: Minimum level to log (e.g. `zerolog.InfoLevel`)

**Returns:**

- A `Logger` that writes to stdout and to daily-rotated files

**Note:** `NewZerologFileLogger` panics if the log directory cannot be created or the initial file writer cannot be set up. Call it at startup and handle panics or validate the path beforehand.

## Basic Usage

### Log Levels

Use `Debug`, `Info`, `Warn`, and `Error` with a message and optional structured fields:

```go
log.Debug("cache hit", logger.Field{Key: "key", Value: "user:123"})
log.Info("request completed", logger.Field{Key: "status", Value: 200}, logger.Field{Key: "ms", Value: 42})
log.Warn("retry attempt", logger.Field{Key: "attempt", Value: 3})
log.Error("database connection failed", logger.Field{Key: "error", Value: err.Error()})
```

**Parameters (for each level method):**

- **msg**: The log message
- **fields**: Optional variadic `logger.Field` key-value pairs to include in the log entry

### Structured Fields (Field)

Attach context with `Field`:

```go
type Field struct {
    Key   string
    Value any
}
```

Examples:

```go
log.Info("user login",
    logger.Field{Key: "user_id", Value: 123},
    logger.Field{Key: "ip", Value: "192.168.1.1"},
)

// Helper can make this shorter in your codebase
func F(k string, v any) logger.Field { return logger.Field{Key: k, Value: v} }
log.Info("order created", F("order_id", id), F("amount", total))
```

### Derived Loggers (With)

Use `With` to create a child logger that includes the given fields in every subsequent log entry. Useful for request IDs, trace IDs, or component names:

```go
requestLog := log.With(
    logger.Field{Key: "request_id", Value: "abc-123"},
    logger.Field{Key: "path", Value: "/api/users"},
)

requestLog.Info("started")
requestLog.Info("completed") // both entries include request_id and path
```

**Parameters:**

- **fields**: Key-value pairs to attach to the derived logger

**Returns:**

- A new `Logger` that includes the specified fields; the original logger is unchanged

### Getting the Underlying zerolog.Logger

For advanced configuration or integration with libraries that accept `zerolog.Logger`, use `GetLoggerInstance()`:

```go
if zl, ok := log.GetLoggerInstance().(zerolog.Logger); ok {
    // use zl for zerolog-specific APIs
}
```

### Closing the Logger

If the logger writes to files (created via `NewZerologFileLogger`), call `Close()` to release file handles. It is safe to call multiple times.

```go
defer log.Close()
```

**Returns:**

- An error if closing resources fails (e.g. flushing/closing the log file)

## Daily File Writer (Advanced)

When you use `NewZerologFileLogger`, the logger uses a `DailyFileWriter` internally. You can also create and use `DailyFileWriter` directly if you need custom wiring (e.g. different output format or only file output).

### NewDailyFileWriter

Creates an `io.Writer` that writes to `{service}_{date}.log` in the given directory. Files rotate automatically at day boundaries; a background goroutine also checks hourly.

```go
import "github.com/cyberinferno/go-utils/logger"

w, err := logger.NewDailyFileWriter("my-service", "/var/log/app")
if err != nil {
    log.Fatal(err)
}
defer w.Close()

// Use as io.Writer (e.g. with zerolog)
zlog := zerolog.New(w)
```

**Parameters:**

- **service**: Service name used in log file names
- **logDir**: Directory path for log files (must exist; not created by this function)

**Returns:**

- The new `*DailyFileWriter`, or an error if the initial file could not be opened

### ForceRotate

Closes the current log file and opens a new one for the current date. Useful when you receive a signal (e.g. SIGHUP) to rotate logs without restarting the process.

```go
if err := fileWriter.ForceRotate(); err != nil {
    log.Printf("force rotate failed: %v", err)
}
```

### CurrentLogFile

Returns the full path of the log file currently being written to, or an empty string if no file is open.

```go
path := fileWriter.CurrentLogFile()
fmt.Println("Logging to:", path) // e.g. /var/log/app/my-service_2026-02-10.log
```

## Usage Examples

### Example 1: Service with File and Console Logging

```go
package main

import (
    "github.com/cyberinferno/go-utils/logger"
    "github.com/rs/zerolog"
)

func main() {
    log := logger.NewZerologFileLogger("my-api", "./logs", zerolog.InfoLevel)
    defer log.Close()

    log.Info("service starting")
    log.Info("listening", logger.Field{Key: "port", Value: 8080})
}
```

### Example 2: Request-Scoped Logger

```go
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    requestID := r.Header.Get("X-Request-ID")
    if requestID == "" {
        requestID = generateID()
    }

    reqLog := h.log.With(
        logger.Field{Key: "request_id", Value: requestID},
        logger.Field{Key: "method", Value: r.Method},
        logger.Field{Key: "path", Value: r.URL.Path},
    )

    reqLog.Info("request started")
    defer func() { reqLog.Info("request completed") }()

    // ... handle request
}
```

### Example 3: Component Logger

```go
type Worker struct {
    log logger.Logger
}

func NewWorker(parent logger.Logger, name string) *Worker {
    return &Worker{
        log: parent.With(logger.Field{Key: "component", Value: name}),
    }
}

func (w *Worker) Run() {
    w.log.Info("worker started")
    w.log.Debug("processing item", logger.Field{Key: "id", Value: 42})
}
```

### Example 4: Using GetLoggerInstance with zerolog

```go
log := logger.NewZerologFileLogger("app", "./logs", zerolog.DebugLevel)
defer log.Close()

zl := log.GetLoggerInstance().(zerolog.Logger)
zl.UpdateContext(func(c zerolog.Context) zerolog.Context {
    return c.Str("version", "1.0.0")
})

log.Info("started") // includes version in context if zerolog preserves it
```

### Example 5: Console-Only with Custom zerolog Output

```go
import (
    "os"
    "github.com/cyberinferno/go-utils/logger"
    "github.com/rs/zerolog"
)

// Pretty console output for development
zlog := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout})
log := logger.NewZerologLogger(zlog, "my-service", zerolog.DebugLevel)

log.Debug("debug message")
log.Info("info message")
```

## Best Practices

### 1. Set Appropriate Log Level

Use `zerolog.DebugLevel` in development and `zerolog.InfoLevel` or `zerolog.WarnLevel` in production to reduce noise and cost:

```go
level := zerolog.InfoLevel
if os.Getenv("DEBUG") == "1" {
    level = zerolog.DebugLevel
}
log := logger.NewZerologFileLogger("my-service", "/var/log/app", level)
```

### 2. Use With() for Request or Component Context

Attach request ID, trace ID, or component name once and have them on every log line:

```go
reqLog := log.With(
    logger.Field{Key: "request_id", Value: requestID},
)
reqLog.Info("started")
reqLog.Info("done") // both lines have request_id
```

### 3. Close the Logger on Shutdown

If you use file logging, defer `Close()` so files are flushed and handles released:

```go
log := logger.NewZerologFileLogger(serviceName, logDir, level)
defer log.Close()
```

### 4. Use Structured Fields Instead of String Formatting

Prefer fields for structured data so log aggregators can index and query them:

```go
// Prefer
log.Info("user login", logger.Field{Key: "user_id", Value: userID})

// Avoid embedding everything in the message
log.Info(fmt.Sprintf("user %d logged in", userID))
```

### 5. Panic Handling for NewZerologFileLogger

`NewZerologFileLogger` panics on setup failure. Call it at startup and either recover or validate paths first:

```go
logDir := os.Getenv("LOG_DIR")
if logDir == "" {
    logDir = "./logs"
}
log := logger.NewZerologFileLogger("my-service", logDir, zerolog.InfoLevel)
defer log.Close()
```

## Type Reference

### Logger Interface

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    With(fields ...Field) Logger
    GetLoggerInstance() interface{}
    Close() error
}
```

### Field

```go
type Field struct {
    Key   string
    Value any
}
```

### NewZerologLogger

```go
func NewZerologLogger(l zerolog.Logger, serviceName string, level zerolog.Level) Logger
```

Builds a Logger that wraps the given zerolog.Logger with service name and timestamp; output goes only to that logger.

### NewZerologFileLogger

```go
func NewZerologFileLogger(serviceName string, logDir string, level zerolog.Level) Logger
```

Creates a Logger that writes to stdout and daily-rotated files. Panics if the directory or initial file cannot be created.

### NewDailyFileWriter

```go
func NewDailyFileWriter(service string, logDir string) (*DailyFileWriter, error)
```

Creates an `io.Writer` that writes to daily-rotated log files. The directory must already exist.

### DailyFileWriter (selected methods)

- **Write(p []byte) (int, error)** — Implements `io.Writer`; rotates when the date changes.
- **Close() error** — Stops the background rotator and closes the current file.
- **ForceRotate() error** — Rotates to a new file immediately (e.g. on SIGHUP).
- **CurrentLogFile() string** — Returns the full path of the current log file, or `""`.

## Error Handling and Panics

### NewZerologFileLogger

- **Panics** if `os.MkdirAll(logDir, 0755)` fails.
- **Panics** if `NewDailyFileWriter(serviceName, logDir)` returns an error (e.g. directory missing when `MkdirAll` was skipped in a custom flow, or permission issues).

Ensure the process has write access to `logDir` and that the path is valid before calling.

### NewDailyFileWriter

- **Returns an error** if the initial log file cannot be opened (e.g. permission denied, read-only filesystem). The directory is not created by this function; create it beforehand.

### DailyFileWriter.Write

- **Returns an error** if the writer has been closed or if rotation or write fails.

### Close()

- **Returns an error** if closing the underlying file fails. Safe to call multiple times; subsequent calls return `nil` after the first successful close.

## Log Levels (zerolog)

Common level values from `github.com/rs/zerolog`:

- `zerolog.TraceLevel` — most verbose
- `zerolog.DebugLevel`
- `zerolog.InfoLevel`
- `zerolog.WarnLevel`
- `zerolog.ErrorLevel`
- `zerolog.FatalLevel` — logs and then exits (use with care)
- `zerolog.Disabled` — no logging

Set the level when creating the logger; messages below that level are not written.
