package main

import (
	"fmt"
	"os"

	"github.com/mtgo-labs/mtgo/telegram"
)

func main() {
	log := telegram.NewLogger("demo")
	log.SetLevel(telegram.TraceLevel)
	log.NoColor(false)

	fmt.Println("=== Basic log levels ===")
	log.Trace("tracing MTProto frames")
	log.Debug("connection established to DC2")
	log.Info("session created successfully")
	log.Warn("flood wait detected, slowing down")
	log.Error("RPC timeout after 3 retries")

	fmt.Println("\n=== Caller location (file:line) ===")
	log.Info("this line shows where the log was called")

	fmt.Println("\n=== Error with root cause ===")
	inner := fmt.Errorf("connection refused")
	middle := fmt.Errorf("dial failed: %w", inner)
	outer := fmt.Errorf("rpc invoke: %w", middle)

	log.ErrorWithCause(outer, "SendMessage failed")
	log.ErrorfWithCause("step %d failed", outer, 3)

	fmt.Println("\n=== Cloned subsystem loggers ===")
	sessionLog := log.Clone("session")
	authLog := log.Clone("auth")

	sessionLog.Info("connected")
	authLog.Debug("phone code requested")

	fmt.Println("\n=== File output + rotation ===")
	logPath := "demo.log"
	if err := log.SetFile(logPath, 1024); err != nil {
		fmt.Printf("set file: %v\n", err)
		return
	}
	log.Info("this line goes to both stderr and the log file")
	log.ErrorWithCause(fmt.Errorf("write failed: %w", fmt.Errorf("disk full")), "cannot save session")
	log.Close()

	data, _ := os.ReadFile(logPath)
	fmt.Printf("\n--- contents of %s ---\n%s\n", logPath, string(data))
	os.Remove(logPath)

	fmt.Println("\n=== Level filtering ===")
	log2 := telegram.NewLogger("filtered")
	log2.SetLevel(telegram.ErrorLevel)
	log2.Debug("this is suppressed")
	log2.Info("this is also suppressed")
	log2.Warn("suppressed too")
	log2.Error("only errors and above pass through")

	fmt.Println("\n=== Done ===")
}
