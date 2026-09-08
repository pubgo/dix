package logger

import (
	"log"

	"github.com/pubgo/dix/v2"
)

// Logger 全局日志接口。
type Logger interface {
	Info(msg string)
	Error(msg string)
}

// ConsoleLogger 默认实现。
type ConsoleLogger struct{ Prefix string }

func (c *ConsoleLogger) Info(msg string)  { log.Printf("[INFO] %s", msg) }
func (c *ConsoleLogger) Error(msg string) { log.Printf("[ERROR] %s", msg) }

func Provide(di *dix.Dix) {
	dix.Provide(di, func() Logger {
		return &ConsoleLogger{Prefix: "app"}
	})
}
