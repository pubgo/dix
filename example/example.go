package main

import (
	"encoding/json"
	"fmt"
	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
	"log"
	"os"
	"time"
)

type Hello interface {
	Hello()
}

type test1 struct {
	h Hello
}

type Config struct {
	Prefix string
}

func (Config) Hello() {
	fmt.Println("config Hello")
}

func init() {
	xerror.Exit(dix.Dix(func(h *test1) {
		h.h.Hello()
	}))
	xerror.Exit(dix.Dix(func(cfg *Config) (*log.Logger, error) {
		return log.New(os.Stdout, cfg.Prefix, log.Llongfile), nil
	}))
	xerror.Exit(dix.Dix(func(l *log.Logger) {
		l.Print("You've been invoked")
	}))
}

func main() {
	i := 0
	for {
		var cfg Config
		xerror.Exit(json.Unmarshal([]byte(fmt.Sprintf(`{"prefix": "[foo%d] "}`, i)), &cfg))
		xerror.Exit(dix.Dix(&cfg))
		xerror.Exit(dix.Dix(&test1{h: &cfg}))
		fmt.Println(dix.Graph())
		time.Sleep(time.Second)
		i++
	}
}
