package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pubgo/dix"
	"github.com/pubgo/funk"
)

type Redis struct {
	name string
}

type Handler struct {
	Cli  *Redis
	Cli1 map[string]*Redis
}

func main() {
	defer funk.RecoverAndExit()

	defer func() {
		fmt.Println(dix.Graph())
	}()

	dix.Provider(func() *log.Logger {
		return log.New(os.Stderr, "example: ", log.LstdFlags|log.Lshortfile)
	})

	dix.Provider(func(p struct {
		L *log.Logger
	}) *Redis {
		p.L.Println("init redis")
		return &Redis{name: "hello"}
	})

	dix.Provider(func(l *log.Logger) map[string]*Redis {
		l.Println("init redis")
		return map[string]*Redis{
			"ns": {name: "hello1"},
		}
	})

	fmt.Println(dix.Graph())

	dix.Inject(func(r *Redis, l *log.Logger, rr map[string]*Redis) {
		l.Println("invoke redis")
		fmt.Println("invoke:", r.name)
		fmt.Println("invoke:", rr)
	})

	var h Handler
	dix.Inject(&h)
	funk.Assert(h.Cli.name != "hello", "inject error")
	funk.Assert(h.Cli1["ns"].name != "hello1", "inject error")

	dix.Inject(func(h Handler) {
		funk.Assert(h.Cli.name != "hello", "inject error")
		funk.Assert(h.Cli1["ns"].name != "hello1", "inject error")
	})
}
