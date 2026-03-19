package dixinternal

import (
	"context"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/lmittmann/tint"
)

var logger = createDefaultLogger()

func getLogPackage() string { return "dix" }

func createDefaultLogger() *slog.Logger {
	logOpt := &tint.Options{Level: slog.LevelInfo, AddSource: true}
	return slog.New(tint.NewHandler(os.Stderr, logOpt)).WithGroup(getLogPackage())
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
