// Package logger provides a structured logging interface with zerolog-backed
// implementations, including optional daily file rotation for persistent logs.
package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

// Field represents a key-value pair for structured log output.
// Use Fields with Logger methods to attach contextual data to log entries.
type Field struct {
	Key   string
	Value any
}

// Logger is an interface for structured logging. Implementations write log
// entries at different levels (Debug, Info, Warn, Error) and support
// attaching structured fields. Loggers may be derived with With for
// request-scoped or component-scoped fields.
type Logger interface {
	// Debug logs a message at debug level with optional structured fields.
	//
	// Parameters:
	//   - msg: The log message
	//   - fields: Optional key-value pairs to include in the log entry
	Debug(msg string, fields ...Field)

	// Info logs a message at info level with optional structured fields.
	//
	// Parameters:
	//   - msg: The log message
	//   - fields: Optional key-value pairs to include in the log entry
	Info(msg string, fields ...Field)

	// Warn logs a message at warn level with optional structured fields.
	//
	// Parameters:
	//   - msg: The log message
	//   - fields: Optional key-value pairs to include in the log entry
	Warn(msg string, fields ...Field)

	// Error logs a message at error level with optional structured fields.
	//
	// Parameters:
	//   - msg: The log message
	//   - fields: Optional key-value pairs to include in the log entry
	Error(msg string, fields ...Field)

	// With returns a new Logger that includes the given fields in all
	// subsequent log entries. The original Logger is unchanged.
	//
	// Parameters:
	//   - fields: Key-value pairs to attach to the derived logger
	//
	// Returns:
	//   - A new Logger with the specified fields
	With(fields ...Field) Logger

	// GetLoggerInstance returns the underlying logger implementation (e.g.
	// zerolog.Logger) for advanced configuration or integration.
	//
	// Returns:
	//   - The underlying logger instance as interface{}
	GetLoggerInstance() interface{}

	// Close releases resources held by the logger (e.g. file handles).
	// It is safe to call multiple times.
	//
	// Returns:
	//   - An error if closing resources fails
	Close() error
}

// zerologLogger is the zerolog-based implementation of Logger.
type zerologLogger struct {
	logger         zerolog.Logger
	fileWriter     *DailyFileWriter
	ownsFileWriter bool
}

// NewZerologLogger builds a Logger that wraps the given zerolog.Logger,
// adding a service name and timestamp to all entries and filtering by level.
// Output goes only to the provided logger (e.g. stdout); no file is created.
//
// Parameters:
//   - l: The zerolog.Logger to wrap
//   - serviceName: Name of the service, added as a field to every log entry
//   - level: Minimum level to log (e.g. zerolog.InfoLevel)
//
// Returns:
//   - A Logger that writes through the given zerolog instance
func NewZerologLogger(l zerolog.Logger, serviceName string, level zerolog.Level) Logger {
	return &zerologLogger{
		logger:         l.With().Str("service", serviceName).Timestamp().Logger().Level(level),
		ownsFileWriter: false,
	}
}

// NewZerologFileLogger creates a Logger that writes to both stdout and
// daily-rotated log files in logDir. Log files are named {serviceName}_{date}.log.
// Panics if logDir cannot be created or the initial file writer cannot be set up.
//
// Parameters:
//   - serviceName: Name of the service, used in log entries and file names
//   - logDir: Directory for log files; created if it does not exist
//   - level: Minimum level to log (e.g. zerolog.InfoLevel)
//
// Returns:
//   - A Logger that writes to stdout and rotating files
func NewZerologFileLogger(serviceName string, logDir string, level zerolog.Level) Logger {
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		panic(fmt.Errorf("failed to create log directory: %w", err))
	}

	fileWriter, err := NewDailyFileWriter(serviceName, logDir)
	if err != nil {
		panic(fmt.Errorf("failed to create file writer: %w", err))
	}

	multi := io.MultiWriter(os.Stdout, fileWriter)
	return &zerologLogger{
		logger:         zerolog.New(multi).With().Str("service", serviceName).Timestamp().Logger().Level(level),
		fileWriter:     fileWriter,
		ownsFileWriter: true,
	}
}

// Debug implements Logger.
func (z *zerologLogger) Debug(msg string, fields ...Field) {
	z.logger.Debug().Fields(toMap(fields)).Msg(msg)
}

// Info implements Logger.
func (z *zerologLogger) Info(msg string, fields ...Field) {
	z.logger.Info().Fields(toMap(fields)).Msg(msg)
}

// Warn implements Logger.
func (z *zerologLogger) Warn(msg string, fields ...Field) {
	z.logger.Warn().Fields(toMap(fields)).Msg(msg)
}

// Error implements Logger.
func (z *zerologLogger) Error(msg string, fields ...Field) {
	z.logger.Error().Fields(toMap(fields)).Msg(msg)
}

// With implements Logger.
func (z *zerologLogger) With(fields ...Field) Logger {
	return &zerologLogger{
		logger:         z.logger.With().Fields(toMap(fields)).Logger(),
		fileWriter:     z.fileWriter,
		ownsFileWriter: false,
	}
}

// GetLoggerInstance implements Logger.
func (z *zerologLogger) GetLoggerInstance() interface{} {
	return z.logger
}

// toMap converts a slice of Field into a map for zerolog.
func toMap(fields []Field) map[string]any {
	if len(fields) == 0 {
		return nil
	}

	m := make(map[string]any, len(fields))
	for _, f := range fields {
		m[f.Key] = f.Value
	}

	return m
}

// Close implements Logger.
func (z *zerologLogger) Close() error {
	if z.fileWriter != nil && z.ownsFileWriter {
		return z.fileWriter.Close()
	}

	return nil
}

// DailyFileWriter is an io.Writer that writes to a log file that rotates
// daily. File names are {service}_{date}.log. Rotation happens automatically
// at midnight and on the first write of a new day; a background goroutine
// also checks hourly. Safe for concurrent use.
type DailyFileWriter struct {
	service    string
	dir        string
	mu         sync.RWMutex
	file       *os.File
	currDate   string
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	closed     int32
	lastRotate time.Time
}

// NewDailyFileWriter creates a DailyFileWriter that writes to the given
// directory with files named {service}_{date}.log. The directory is not
// created by this function; callers must ensure it exists.
//
// Parameters:
//   - service: Service name used in log file names
//   - logDir: Directory path for log files
//
// Returns:
//   - The new DailyFileWriter, or an error if the initial file could not be opened
func NewDailyFileWriter(service string, logDir string) (*DailyFileWriter, error) {
	ctx, cancel := context.WithCancel(context.Background())
	w := &DailyFileWriter{
		service: service,
		dir:     logDir,
		ctx:     ctx,
		cancel:  cancel,
	}

	if err := w.rotate(); err != nil {
		cancel()
		return nil, fmt.Errorf("initial rotation failed: %w", err)
	}

	w.wg.Add(1)
	go w.autoRotate()
	return w, nil
}

// Close stops the background rotator and closes the current log file.
// Subsequent writes return an error. It is safe to call multiple times.
//
// Returns:
//   - An error if closing the file fails
func (w *DailyFileWriter) Close() error {
	if !atomic.CompareAndSwapInt32(&w.closed, 0, 1) {
		return nil
	}

	w.cancel()
	w.wg.Wait()

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		err := w.file.Close()
		w.file = nil
		return err
	}

	return nil
}

// autoRotate runs in a goroutine and performs hourly rotation checks.
func (w *DailyFileWriter) autoRotate() {
	defer w.wg.Done()

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			if atomic.LoadInt32(&w.closed) == 1 {
				return
			}

			w.mu.Lock()
			_ = w.rotateInternal()
			w.mu.Unlock()
		}
	}
}

// rotate switches to a new log file if the date has changed. It is safe to call concurrently.
//
// Returns:
//   - An error if the writer is closed or the new file could not be opened
func (w *DailyFileWriter) rotate() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.rotateInternal()
}

// rotateInternal performs the actual file rotation; caller must hold w.mu.
func (w *DailyFileWriter) rotateInternal() error {
	if atomic.LoadInt32(&w.closed) == 1 {
		return fmt.Errorf("writer is closed")
	}

	now := time.Now()
	date := now.Format("2006-01-02")

	if date == w.currDate && w.file != nil &&
		now.Sub(w.lastRotate) < time.Minute {
		return nil
	}

	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}

	filename := filepath.Join(w.dir, fmt.Sprintf("%s_%s.log", w.service, date))
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", filename, err)
	}

	w.file = file
	w.currDate = date
	w.lastRotate = now
	return nil
}

// Write implements io.Writer. It rotates to a new file when the date changes
// and writes p to the current log file.
//
// Returns:
//   - The number of bytes written and an error if the writer is closed or write fails
func (w *DailyFileWriter) Write(p []byte) (int, error) {
	if atomic.LoadInt32(&w.closed) == 1 {
		return 0, fmt.Errorf("writer is closed")
	}

	w.mu.RLock()
	needsRotation := w.needsRotation()
	currentFile := w.file
	w.mu.RUnlock()

	if needsRotation {
		w.mu.Lock()
		if w.needsRotation() {
			if err := w.rotateInternal(); err != nil {
				w.mu.Unlock()
				return 0, fmt.Errorf("rotation failed: %w", err)
			}
		}

		currentFile = w.file
		w.mu.Unlock()
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.file == nil {
		return 0, fmt.Errorf("log file is not open")
	}

	if w.file != currentFile {
		currentFile = w.file
	}

	return currentFile.Write(p)
}

// needsRotation reports whether the log file should be rotated (e.g. new day).
func (w *DailyFileWriter) needsRotation() bool {
	if w.file == nil {
		return true
	}

	date := time.Now().Format("2006-01-02")
	return date != w.currDate
}

// ForceRotate closes the current log file and opens a new one for the current date.
// Useful for external rotation triggers (e.g. SIGHUP).
//
// Returns:
//   - An error if rotation fails
func (w *DailyFileWriter) ForceRotate() error {
	return w.rotate()
}

// CurrentLogFile returns the full path of the log file currently being written to.
// Returns an empty string if no file is open.
//
// Returns:
//   - The path to the current log file, or "" if none
func (w *DailyFileWriter) CurrentLogFile() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.file == nil {
		return ""
	}

	return filepath.Join(w.dir, fmt.Sprintf("%s_%s.log", w.service, w.currDate))
}
