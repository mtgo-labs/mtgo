package telegram

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"
)

func newTestLogger() (*Logger, *bytes.Buffer) {
	l := NewLogger("test")
	l.SetLevel(TraceLevel)
	l.NoColor(true)
	l.exitFn = func(int) {}
	var buf bytes.Buffer
	l.output = log.New(&buf, "", 0)
	return l, &buf
}

func TestFatalLevel(t *testing.T) {
	l, _ := newTestLogger()
	l.SetLevel(FatalLevel)

	if !l.enabled(FatalLevel) {
		t.Error("FatalLevel should be enabled when level is FatalLevel")
	}
	if l.enabled(ErrorLevel) {
		t.Error("ErrorLevel should NOT be enabled when level is FatalLevel")
	}
	if l.enabled(WarnLevel) {
		t.Error("WarnLevel should NOT be enabled when level is FatalLevel")
	}
}

func TestFatalExits(t *testing.T) {
	l, buf := newTestLogger()
	var exitCode int
	l.exitFn = func(code int) { exitCode = code }

	l.Fatal("something broke")

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	out := buf.String()
	if !strings.Contains(out, "FATAL") {
		t.Errorf("expected FATAL in output, got: %s", out)
	}
	if !strings.Contains(out, "something broke") {
		t.Errorf("expected message in output, got: %s", out)
	}
}

func TestFatalfExits(t *testing.T) {
	l, buf := newTestLogger()
	var exitCode int
	l.exitFn = func(code int) { exitCode = code }

	l.Fatalf("code=%d", 42)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	out := buf.String()
	if !strings.Contains(out, "code=42") {
		t.Errorf("expected formatted message, got: %s", out)
	}
}

func TestCallerLocation(t *testing.T) {
	l, buf := newTestLogger()

	l.Error("test location")

	out := buf.String()
	if !strings.Contains(out, "logger_test.go") {
		t.Errorf("expected caller file in output, got: %s", out)
	}
}

func TestErrorWithCause(t *testing.T) {
	l, buf := newTestLogger()

	inner := errors.New("connection refused")
	middle := fmt.Errorf("dial failed: %w", inner)
	outer := fmt.Errorf("rpc error: %w", middle)

	l.ErrorWithCause(outer, "invoke failed")

	out := buf.String()
	if !strings.Contains(out, "invoke failed") {
		t.Errorf("expected message in output, got: %s", out)
	}
	if !strings.Contains(out, "root_cause: connection refused") {
		t.Errorf("expected root_cause in output, got: %s", out)
	}
	if !strings.Contains(out, "rpc error: dial failed: connection refused") {
		t.Errorf("expected full error chain in output, got: %s", out)
	}
}

func TestErrorWithCauseNoWrapping(t *testing.T) {
	l, buf := newTestLogger()

	err := errors.New("simple error")

	l.ErrorWithCause(err, "failed")

	out := buf.String()
	if !strings.Contains(out, "error: simple error") {
		t.Errorf("expected error in output, got: %s", out)
	}
	if strings.Contains(out, "root_cause") {
		t.Errorf("should not show root_cause when no wrapping, got: %s", out)
	}
}

func TestErrorfWithCause(t *testing.T) {
	l, buf := newTestLogger()

	inner := errors.New("timeout")
	wrapped := fmt.Errorf("context: %w", inner)

	l.ErrorfWithCause("step %d failed", wrapped, 3)

	out := buf.String()
	if !strings.Contains(out, "step 3 failed") {
		t.Errorf("expected formatted message, got: %s", out)
	}
	if !strings.Contains(out, "root_cause: timeout") {
		t.Errorf("expected root_cause, got: %s", out)
	}
}

func TestFatalWithCause(t *testing.T) {
	l, buf := newTestLogger()
	var exitCode int
	l.exitFn = func(code int) { exitCode = code }

	inner := errors.New("disk full")
	wrapped := fmt.Errorf("write failed: %w", inner)

	l.FatalWithCause(wrapped, "cannot proceed")

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	out := buf.String()
	if !strings.Contains(out, "root_cause: disk full") {
		t.Errorf("expected root_cause, got: %s", out)
	}
}

func TestRootCause(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect string
	}{
		{
			name:   "simple error",
			err:    errors.New("base"),
			expect: "base",
		},
		{
			name:   "single wrap",
			err:    fmt.Errorf("wrap: %w", errors.New("base")),
			expect: "base",
		},
		{
			name:   "double wrap",
			err:    fmt.Errorf("a: %w", fmt.Errorf("b: %w", errors.New("c"))),
			expect: "c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rootCause(tt.err)
			if got != tt.expect {
				t.Errorf("rootCause() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestCloneInheritsSettings(t *testing.T) {
	l, _ := newTestLogger()
	child := l.Clone("subsystem")

	if child.prefix != "test subsystem" {
		t.Errorf("expected prefix 'test subsystem', got %q", child.prefix)
	}
}

func TestConcurrentLogging(t *testing.T) {
	l, _ := newTestLogger()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			l.Info("msg", n)
			l.Error("err", n)
			l.Debug("dbg", n)
		}(i)
	}
	wg.Wait()
}

func TestAllLevelsOrdered(t *testing.T) {
	if TraceLevel >= DebugLevel {
		t.Error("TraceLevel should be < DebugLevel")
	}
	if DebugLevel >= InfoLevel {
		t.Error("DebugLevel should be < InfoLevel")
	}
	if InfoLevel >= WarnLevel {
		t.Error("InfoLevel should be < WarnLevel")
	}
	if WarnLevel >= ErrorLevel {
		t.Error("WarnLevel should be < ErrorLevel")
	}
	if ErrorLevel >= FatalLevel {
		t.Error("ErrorLevel should be < FatalLevel")
	}
	if FatalLevel >= NoLevel {
		t.Error("FatalLevel should be < NoLevel")
	}
}
