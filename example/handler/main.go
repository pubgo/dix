package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
)

type Redis struct {
	name string
}

type Handler struct {
	name string
	Cli  *Redis
	Cli1 map[string]*Redis
}

func main() {
	defer xerror.RecoverAndExit()

	dix.Register(func() *log.Logger {
		return log.New(os.Stderr, "example: ", log.LstdFlags|log.Lshortfile)
	})

	dix.Register(func(l *log.Logger) *Redis {
		l.Println("init redis")
		return &Redis{name: "hello"}
	})

	dix.Register(func(l *log.Logger) map[string]*Redis {
		l.Println("init redis")
		return map[string]*Redis{
			"ns": {name: "hello1"},
		}
	})

	dix.Inject(func(r *Redis, l *log.Logger, rr map[string]*Redis) {
		l.Println("invoke redis")
		fmt.Println("invoke:", r.name)
		fmt.Println("invoke:", rr)
	})

	var h = Handler{name: "ns"}
	dix.Inject(&h)
	xerror.Assert(h.Cli.name != "hello", "inject error")
	xerror.Assert(h.Cli1["ns"].name != "hello1", "inject error")
	fmt.Println(dix.Graph())
}
