package telegram

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the verbosity level for log filtering. Messages with a
// level below the configured threshold are silently discarded.
type LogLevel int

const (
	// TraceLevel is the most verbose level, intended for tracing MTProto
	// frame-level activity such as message serialization and RPC round-trips.
	// Enable only during deep debugging due to high output volume.
	TraceLevel LogLevel = iota

	// DebugLevel is used for general debugging information such as connection
	// lifecycle events, reconnection attempts, and RPC request/response pairs.
	DebugLevel

	// InfoLevel is the standard operational level for noteworthy events like
	// successful connection establishment, DC migration, and session creation.
	InfoLevel

	// WarnLevel indicates unexpected but recoverable conditions such as
	// deprecated API usage, retryable RPC errors, or slow responses.
	WarnLevel

	// ErrorLevel reports failures that require attention: authentication errors,
	// persistent connection drops, or unrecoverable RPC failures.
	ErrorLevel

	// NoLevel disables all log output. This is the default level for new Logger
	// instances so that a freshly created Client produces no output until a level
	// is explicitly set.
	NoLevel
)

var defaultMaxSize int64 = 10 * 1024 * 1024

var (
	logColorOff    = "\033[0m"
	logColorRed    = "\033[0;31m"
	logColorGreen  = "\033[0;32m"
	logColorYellow = "\033[0;33m"
	logColorPurple = "\033[0;35m"
	logColorCyan   = "\033[0;36m"
	logColorGray   = "\033[0;90m"
)

// Logger provides leveled, optionally colorized logging for the MTProto client.
// It supports concurrent use, file output with automatic rotation, and prefix
// chaining via Clone for subsystem-scoped loggers (e.g. "session", "auth").
type Logger struct {
	mu     sync.Mutex
	prefix string
	// Level controls the minimum severity that will be emitted. Set via
	// SetLevel. Messages at or above this level are printed; those below are
	// silently discarded. NoLevel (the default) disables all output.
	Level    LogLevel
	noColor  bool
	output   *log.Logger
	file     *os.File
	filePath string
	maxSize  int64
}

// NewLogger creates a Logger that writes to stderr with the given prefix.
// The initial level is NoLevel (all output suppressed) so callers must call
// SetLevel to enable output.
//
// Parameters:
//   - prefix: label included in every log line, typically a subsystem name
//     such as "auth" or "rpc".
//
// Returns:
//   - *Logger ready for use (level defaults to NoLevel).
func NewLogger(prefix string) *Logger {
	return &Logger{
		prefix:  prefix,
		Level:   NoLevel,
		output:  log.New(os.Stderr, "", 0),
		maxSize: defaultMaxSize,
	}
}

// NoColor controls ANSI color output. Pass true (or no argument) to disable
// color; pass false to re-enable it. Returns the receiver for chaining.
//
// Parameters:
//   - v: optional single bool; true disables color (default when omitted).
//
// Returns:
//   - *Logger: the receiver, for method chaining.
func (l *Logger) NoColor(v ...bool) *Logger {
	l.noColor = true
	if len(v) > 0 {
		l.noColor = v[0]
	}
	return l
}

// SetLevel sets the minimum log severity. Messages below this level are
// discarded. Returns the receiver for chaining.
//
// Parameters:
//   - level: the minimum LogLevel to emit.
//
// Returns:
//   - *Logger: the receiver, for method chaining.
func (l *Logger) SetLevel(level LogLevel) *Logger {
	l.Level = level
	return l
}

// SetPrefix replaces the logger's prefix string. Returns the receiver for
// chaining.
//
// Parameters:
//   - prefix: new prefix label for all subsequent log lines.
//
// Returns:
//   - *Logger: the receiver, for method chaining.
func (l *Logger) SetPrefix(prefix string) *Logger {
	l.prefix = prefix
	return l
}

// SetFile directs log output to both stderr and the named file. The file is
// opened in append mode and the parent directory is created if necessary.
// When the file exceeds maxSize bytes it is rotated: the existing file is
// renamed with a timestamp suffix and a new file is created.
//
// Parameters:
//   - path: filesystem path for the log file.
//   - maxSize: maximum size in bytes before rotation. Pass 0 or a negative
//     value to use the default (10 MB).
//
// Returns:
//   - error: if the directory cannot be created or the file cannot be opened.
func (l *Logger) SetFile(path string, maxSize int64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.file.Close()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	if maxSize <= 0 {
		maxSize = defaultMaxSize
	}
	l.maxSize = maxSize
	l.filePath = path

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	l.file = f
	l.output = log.New(io.MultiWriter(os.Stderr, f), "", 0)
	return nil
}

// Close flushes and closes the log file if one is open. After closing, output
// reverts to stderr only.
//
// Returns:
//   - error: if the underlying file close fails, or nil if no file was open.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

// Clone creates a child Logger that inherits the parent's level, file output,
// and color settings but prepends the parent's prefix to the new prefix.
// Useful for creating subsystem-scoped loggers (e.g. "session rpc").
//
// Parameters:
//   - prefix: additional prefix segment appended to the parent's prefix.
//
// Returns:
//   - *Logger: a new Logger sharing the same output destination.
func (l *Logger) Clone(prefix string) *Logger {
	cloned := &Logger{
		prefix:   l.prefix + " " + prefix,
		Level:    l.Level,
		noColor:  l.noColor,
		filePath: l.filePath,
		maxSize:  l.maxSize,
	}
	if l.file != nil {
		cloned.file = l.file
		cloned.output = log.New(io.MultiWriter(os.Stderr, l.file), "", 0)
	} else {
		cloned.output = log.New(os.Stderr, "", 0)
	}
	return cloned
}

func (l *Logger) colorize(color string, s string) string {
	if l.noColor {
		return s
	}
	return color + s + logColorOff
}

func (l *Logger) enabled(level LogLevel) bool {
	if l == nil {
		return false
	}
	return level >= l.Level && l.Level != NoLevel
}

func (l *Logger) maybeRotate() {
	if l.filePath == "" || l.file == nil || l.maxSize <= 0 {
		return
	}
	info, err := l.file.Stat()
	if err != nil {
		return
	}
	if info.Size() < l.maxSize {
		return
	}
	l.file.Close()
	ts := time.Now().Format("20060102-150405")
	ext := filepath.Ext(l.filePath)
	base := strings.TrimSuffix(l.filePath, ext)
	backup := fmt.Sprintf("%s-%s%s", base, ts, ext)
	os.Rename(l.filePath, backup)
	f, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		l.output = log.New(os.Stderr, "", 0)
		l.file = nil
		return
	}
	l.file = f
	l.output = log.New(io.MultiWriter(os.Stderr, f), "", 0)
}

func (l *Logger) log(level LogLevel, color, tag string, v ...any) {
	if !l.enabled(level) {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	l.maybeRotate()

	ts := time.Now().Format("15:04:05.000")
	msg := formatArgs(v...)

	sb := strings.Builder{}
	sb.WriteString(l.colorize(logColorGray, ts))
	sb.WriteString(" ")
	sb.WriteString(l.colorize(color, tag))
	sb.WriteString(" ")
	sb.WriteString(l.colorize(logColorCyan, "["+l.prefix+"]"))
	sb.WriteString(" ")
	sb.WriteString(msg)
	l.output.Println(sb.String())
}

// Debug logs one or more values at DebugLevel. Arguments are concatenated with
// spaces, similar to fmt.Println.
//
// Parameters:
//   - v: values to log; printed as with fmt.Sprint.
func (l *Logger) Debug(v ...any) {
	l.log(DebugLevel, logColorPurple, "DEBUG", v...)
}

// Debugf logs a formatted message at DebugLevel.
//
// Parameters:
//   - format: fmt.Sprintf format string.
//   - v: format arguments.
func (l *Logger) Debugf(format string, v ...any) {
	if !l.enabled(DebugLevel) {
		return
	}
	l.log(DebugLevel, logColorPurple, "DEBUG", fmt.Sprintf(format, v...))
}

// Info logs one or more values at InfoLevel.
//
// Parameters:
//   - v: values to log; printed as with fmt.Sprint.
func (l *Logger) Info(v ...any) {
	l.log(InfoLevel, logColorGreen, "INFO", v...)
}

// Infof logs a formatted message at InfoLevel.
//
// Parameters:
//   - format: fmt.Sprintf format string.
//   - v: format arguments.
func (l *Logger) Infof(format string, v ...any) {
	if !l.enabled(InfoLevel) {
		return
	}
	l.log(InfoLevel, logColorGreen, "INFO", fmt.Sprintf(format, v...))
}

// Warn logs one or more values at WarnLevel.
//
// Parameters:
//   - v: values to log; printed as with fmt.Sprint.
func (l *Logger) Warn(v ...any) {
	l.log(WarnLevel, logColorYellow, "WARN", v...)
}

// Warnf logs a formatted message at WarnLevel.
//
// Parameters:
//   - format: fmt.Sprintf format string.
//   - v: format arguments.
func (l *Logger) Warnf(format string, v ...any) {
	if !l.enabled(WarnLevel) {
		return
	}
	l.log(WarnLevel, logColorYellow, "WARN", fmt.Sprintf(format, v...))
}

// Error logs one or more values at ErrorLevel.
//
// Parameters:
//   - v: values to log; printed as with fmt.Sprint.
func (l *Logger) Error(v ...any) {
	l.log(ErrorLevel, logColorRed, "ERROR", v...)
}

// Errorf logs a formatted message at ErrorLevel.
//
// Parameters:
//   - format: fmt.Sprintf format string.
//   - v: format arguments.
func (l *Logger) Errorf(format string, v ...any) {
	if !l.enabled(ErrorLevel) {
		return
	}
	l.log(ErrorLevel, logColorRed, "ERROR", fmt.Sprintf(format, v...))
}

// Trace logs one or more values at TraceLevel. This is the most verbose level
// and should only be enabled when diagnosing low-level protocol issues.
//
// Parameters:
//   - v: values to log; printed as with fmt.Sprint.
func (l *Logger) Trace(v ...any) {
	l.log(TraceLevel, logColorCyan, "TRACE", v...)
}

// Tracef logs a formatted message at TraceLevel.
//
// Parameters:
//   - format: fmt.Sprintf format string.
//   - v: format arguments.
func (l *Logger) Tracef(format string, v ...any) {
	if !l.enabled(TraceLevel) {
		return
	}
	l.log(TraceLevel, logColorCyan, "TRACE", fmt.Sprintf(format, v...))
}

func formatArgs(v ...any) string {
	if len(v) == 0 {
		return ""
	}
	if len(v) == 1 {
		return fmt.Sprint(v[0])
	}
	return strings.Trim(fmt.Sprint(v...), " ")
}
