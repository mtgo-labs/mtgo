package telegram

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type LogLevel int

const (
	TraceLevel LogLevel = iota
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	NoLevel
)

var defaultMaxSize int64 = 10 * 1024 * 1024

var (
	logColorOff     = "\033[0m"
	logColorRed     = "\033[0;31m"
	logColorBoldRed = "\033[1;31m"
	logColorGreen   = "\033[0;32m"
	logColorYellow  = "\033[0;33m"
	logColorPurple  = "\033[0;35m"
	logColorCyan    = "\033[0;36m"
	logColorGray    = "\033[0;90m"
	logColorWhite   = "\033[0;37m"
)

type Logger struct {
	mu       sync.Mutex
	rotateMu *sync.Mutex
	prefix   string
	level    atomic.Int32
	noColor  atomic.Int32
	caller   atomic.Int32
	output   *log.Logger
	file     *os.File
	filePath string
	maxSize  int64
	exitFn   func(int)
}

func NewLogger(prefix string) *Logger {
	l := &Logger{
		prefix:   prefix,
		output:   log.New(os.Stderr, "", 0),
		maxSize:  defaultMaxSize,
		rotateMu: &sync.Mutex{},
		exitFn:   os.Exit,
	}
	l.level.Store(int32(NoLevel))
	return l
}

func (l *Logger) NoColor(v ...bool) *Logger {
	nc := int32(1)
	if len(v) > 0 && !v[0] {
		nc = 0
	}
	l.noColor.Store(nc)
	return l
}

func (l *Logger) SetLevel(level LogLevel) *Logger {
	l.level.Store(int32(level))
	return l
}

func (l *Logger) SetPrefix(prefix string) *Logger {
	l.prefix = prefix
	return l
}

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

func (l *Logger) Clone(prefix string) *Logger {
	cloned := &Logger{
		prefix:   l.prefix + " " + prefix,
		filePath: l.filePath,
		maxSize:  l.maxSize,
		rotateMu: l.rotateMu,
		exitFn:   l.exitFn,
	}
	cloned.level.Store(l.level.Load())
	cloned.noColor.Store(l.noColor.Load())
	cloned.caller.Store(l.caller.Load())
	if l.file != nil {
		cloned.file = l.file
		cloned.output = log.New(io.MultiWriter(os.Stderr, l.file), "", 0)
	} else {
		cloned.output = log.New(os.Stderr, "", 0)
	}
	return cloned
}

func (l *Logger) colorize(color string, s string) string {
	if l.noColor.Load() != 0 {
		return s
	}
	return color + s + logColorOff
}

func (l *Logger) enabled(level LogLevel) bool {
	if l == nil {
		return false
	}
	stored := LogLevel(l.level.Load())
	return level >= stored && stored != NoLevel
}

func (l *Logger) maybeRotate() {
	if l.filePath == "" || l.file == nil || l.maxSize <= 0 {
		return
	}
	l.rotateMu.Lock()
	defer l.rotateMu.Unlock()
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

func callerInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return ""
	}
	dir := filepath.Dir(file)
	pkg := filepath.Base(dir)
	return fmt.Sprintf("%s/%s:%d", pkg, filepath.Base(file), line)
}

func rootCause(err error) string {
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err.Error()
		}
		err = unwrapped
	}
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

	loc := callerInfo(2)
	if loc != "" {
		sb.WriteString(" ")
		sb.WriteString(l.colorize(logColorWhite, loc))
	}

	sb.WriteString(" ")
	sb.WriteString(msg)
	l.output.Println(sb.String())
}

func (l *Logger) logWithCause(level LogLevel, color, tag string, err error, v ...any) {
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

	loc := callerInfo(2)
	if loc != "" {
		sb.WriteString(" ")
		sb.WriteString(l.colorize(logColorWhite, loc))
	}

	sb.WriteString(" ")
	sb.WriteString(msg)

	if err != nil {
		cause := rootCause(err)
		if cause != err.Error() {
			sb.WriteString(" | error: ")
			sb.WriteString(err.Error())
			sb.WriteString(" | root_cause: ")
			sb.WriteString(cause)
		} else {
			sb.WriteString(" | error: ")
			sb.WriteString(cause)
		}
	}

	l.output.Println(sb.String())
}

func (l *Logger) Debug(v ...any) {
	l.log(DebugLevel, logColorPurple, "DEBUG", v...)
}

func (l *Logger) Debugf(format string, v ...any) {
	if !l.enabled(DebugLevel) {
		return
	}
	l.log(DebugLevel, logColorPurple, "DEBUG", fmt.Sprintf(format, v...))
}

func (l *Logger) Info(v ...any) {
	l.log(InfoLevel, logColorGreen, "INFO", v...)
}

func (l *Logger) Infof(format string, v ...any) {
	if !l.enabled(InfoLevel) {
		return
	}
	l.log(InfoLevel, logColorGreen, "INFO", fmt.Sprintf(format, v...))
}

func (l *Logger) Warn(v ...any) {
	l.log(WarnLevel, logColorYellow, "WARN", v...)
}

func (l *Logger) Warnf(format string, v ...any) {
	if !l.enabled(WarnLevel) {
		return
	}
	l.log(WarnLevel, logColorYellow, "WARN", fmt.Sprintf(format, v...))
}

func (l *Logger) Error(v ...any) {
	l.log(ErrorLevel, logColorRed, "ERROR", v...)
}

func (l *Logger) Errorf(format string, v ...any) {
	if !l.enabled(ErrorLevel) {
		return
	}
	l.log(ErrorLevel, logColorRed, "ERROR", fmt.Sprintf(format, v...))
}

func (l *Logger) ErrorWithCause(err error, v ...any) {
	l.logWithCause(ErrorLevel, logColorRed, "ERROR", err, v...)
}

func (l *Logger) ErrorfWithCause(format string, err error, v ...any) {
	if !l.enabled(ErrorLevel) {
		return
	}
	l.logWithCause(ErrorLevel, logColorRed, "ERROR", err, fmt.Sprintf(format, v...))
}

func (l *Logger) Fatal(v ...any) {
	l.log(FatalLevel, logColorBoldRed, "FATAL", v...)
	l.exitFn(1)
}

func (l *Logger) Fatalf(format string, v ...any) {
	if !l.enabled(FatalLevel) {
		l.exitFn(1)
		return
	}
	l.log(FatalLevel, logColorBoldRed, "FATAL", fmt.Sprintf(format, v...))
	l.exitFn(1)
}

func (l *Logger) FatalWithCause(err error, v ...any) {
	l.logWithCause(FatalLevel, logColorBoldRed, "FATAL", err, v...)
	l.exitFn(1)
}

func (l *Logger) Trace(v ...any) {
	l.log(TraceLevel, logColorCyan, "TRACE", v...)
}

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
	return strings.TrimSuffix(fmt.Sprintln(v...), "\n")
}
