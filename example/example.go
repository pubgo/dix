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
	i int
}

func (t test1) Hello() {
	fmt.Println("config test1")
}

type Config struct {
	Prefix string
}

func (Config) Hello() {
	fmt.Println("Hello Config")
}

func init() {
	xerror.Exit(dix.Dix(func(h *test1) {
		fmt.Println("h *test1")
	}))
	xerror.Exit(dix.Dix(func(h Hello) {
		h.Hello()
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
		fmt.Println(dix.Graph())
		fmt.Print("==================================================================================\n")
		time.Sleep(time.Second)
		xerror.Exit(dix.Dix(&test1{i:i}))
		fmt.Println(dix.Graph())
		time.Sleep(time.Second)
		i++
	}
}
