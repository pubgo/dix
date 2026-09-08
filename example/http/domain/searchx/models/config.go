package models

import "github.com/pubgo/dix/v2"

// Config searchx 域配置。
type Config struct {
	Env     string
	Timeout string
}

func Provide(di *dix.Dix) {
	dix.Provide(di, func() *Config {
		return &Config{Env: "prod", Timeout: "3s"}
	})
}
