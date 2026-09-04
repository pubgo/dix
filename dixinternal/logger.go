package dixinternal

import (
	"context"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"

	"github.com/lmittmann/tint"
)

var logger = createDefaultLogger()

const (
	diTraceEnv = "DIX_TRACE_DI"
)

func getLogPackage() string { return "dix" }

func createDefaultLogger() *slog.Logger {
	logOpt := &tint.Options{Level: slog.LevelInfo, AddSource: true}
	return slog.New(tint.NewHandler(os.Stderr, logOpt)).WithGroup(getLogPackage())
}

func shouldTraceDependencyFlow() bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv(diTraceEnv))) {
	case "1", "true", "on", "yes", "y", "enable", "enabled", "trace", "debug":
		return true
	default:
		return false
	}
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
