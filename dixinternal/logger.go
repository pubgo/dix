package dixinternal

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"

	"github.com/lmittmann/tint"
)

var logger = createDefaultLogger()

const (
	llmDiagModeEnv     = "DIX_LLM_DIAG_MODE"
	llmDiagModeHuman   = "human"
	llmDiagModeMachine = "machine"
	llmDiagModeDual    = "dual"
	diTraceEnv         = "DIX_TRACE_DI"
)

func getLogPackage() string { return "dix" }

func createDefaultLogger() *slog.Logger {
	if isLLMDiagMachineOnlyMode() {
		return slog.New(slog.NewTextHandler(io.Discard, nil)).WithGroup(getLogPackage())
	}

	logOpt := &tint.Options{Level: slog.LevelInfo, AddSource: true}
	return slog.New(tint.NewHandler(os.Stderr, logOpt)).WithGroup(getLogPackage())
}

func currentLLMDiagMode() string {
	mode := strings.TrimSpace(strings.ToLower(os.Getenv(llmDiagModeEnv)))
	switch mode {
	case "", "human", "text", "reader":
		return llmDiagModeHuman
	case "only", "machine", "machine-only", "machine_only", "json":
		return llmDiagModeMachine
	case "dual", "both", "default":
		return llmDiagModeDual
	default:
		return llmDiagModeHuman
	}
}

func isLLMDiagMachineOnlyMode() bool {
	return currentLLMDiagMode() == llmDiagModeMachine
}

func shouldEmitLLMDiagnosticLine() bool {
	mode := currentLLMDiagMode()
	return mode == llmDiagModeMachine || mode == llmDiagModeDual
}

func shouldTraceDependencyFlow() bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv(diTraceEnv))) {
	case "1", "true", "on", "yes", "y", "enable", "enabled", "trace", "debug":
		return true
	default:
		return false
	}
}

func logDITrace(event string, args ...any) {
	if !shouldTraceDependencyFlow() || logger == nil {
		return
	}
	logger.Info("di_trace "+event, args...)
}

func SetLog(handler slog.Handler) {
	logger = slog.New(handler).WithGroup(getLogPackage())
}

func shouldPrintStack() bool {
	if logger == nil {
		return false
	}
	return logger.Enabled(context.Background(), slog.LevelDebug)
}

func maybePrintStack() {
	if shouldPrintStack() {
		debug.PrintStack()
	}
}
