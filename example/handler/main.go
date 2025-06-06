package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pubgo/dix/di"
	"github.com/pubgo/funk/assert"
	"github.com/pubgo/funk/recovery"
)

type Redis struct {
	name string
}

type Handler struct {
	Cli  *Redis
	Cli1 map[string]*Redis
}

func main() {
	defer recovery.Exit()

	defer func() {
		fmt.Println(diglobal.Graph())
	}()

	diglobal.Provide(func() *log.Logger {
		return log.New(os.Stderr, "example: ", log.LstdFlags|log.Lshortfile)
	})

	diglobal.Provide(func(p struct {
		L *log.Logger
	},
	) *Redis {
		p.L.Println("init redis")
		return &Redis{name: "hello"}
	})

	diglobal.Provide(func(l *log.Logger) map[string]*Redis {
		l.Println("init redis")
		return map[string]*Redis{
			"ns": {name: "hello1"},
		}
	})

	fmt.Println(diglobal.Graph())

	diglobal.Inject(func(r *Redis, l *log.Logger, rr map[string]*Redis) {
		l.Println("invoke redis")
		fmt.Println("invoke:", r.name)
		fmt.Println("invoke:", rr)
	})

	h := diglobal.Inject(new(Handler))
	assert.If(h.Cli.name != "hello", "inject error")
	assert.If(h.Cli1["ns"].name != "hello1", "inject error")

	diglobal.Inject(func(h Handler) {
		assert.If(h.Cli.name != "hello", "inject error")
		assert.If(h.Cli1["ns"].name != "hello1", "inject error")
	})
}
