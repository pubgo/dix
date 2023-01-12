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
		fmt.Println(di.Graph())
	}()

	di.Provide(func() *log.Logger {
		return log.New(os.Stderr, "example: ", log.LstdFlags|log.Lshortfile)
	})

	di.Provide(func(p struct {
		L *log.Logger
	}) *Redis {
		p.L.Println("init redis")
		return &Redis{name: "hello"}
	})

	di.Provide(func(l *log.Logger) map[string]*Redis {
		l.Println("init redis")
		return map[string]*Redis{
			"ns": {name: "hello1"},
		}
	})

	fmt.Println(di.Graph())

	di.Inject(func(r *Redis, l *log.Logger, rr map[string]*Redis) {
		l.Println("invoke redis")
		fmt.Println("invoke:", r.name)
		fmt.Println("invoke:", rr)
	})

	var h Handler
	di.Inject(&h)
	assert.If(h.Cli.name != "hello", "inject error")
	assert.If(h.Cli1["ns"].name != "hello1", "inject error")

	di.Inject(func(h Handler) {
		assert.If(h.Cli.name != "hello", "inject error")
		assert.If(h.Cli1["ns"].name != "hello1", "inject error")
	})
}
