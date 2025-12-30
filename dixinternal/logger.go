package dixinternal

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

var logger = createDefaultLogger()

func getLogPackage() slog.Attr {
	return slog.String("package", "dix")
}

func createDefaultLogger() *slog.Logger {
	logOpt := &tint.Options{Level: slog.LevelInfo, AddSource: true}
	return slog.New(tint.NewHandler(os.Stderr, logOpt)).With(getLogPackage())
}

func SetLog(handler slog.Handler) {
	logger = slog.New(handler).With(getLogPackage())
}
