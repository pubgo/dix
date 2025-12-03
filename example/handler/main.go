package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pubgo/dix/v2/dixglobal"
)

type Redis struct {
	name string
}

type Handler struct {
	Cli  *Redis
	Cli1 map[string]*Redis
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic: %v\n", r)
		}
	}()

	defer func() {
		fmt.Println(dixglobal.Graph())
	}()

	dixglobal.Provide(func() *log.Logger {
		return log.New(os.Stderr, "example: ", log.LstdFlags|log.Lshortfile)
	})

	dixglobal.Provide(func(p struct {
		L *log.Logger
	},
	) *Redis {
		p.L.Println("init redis")
		return &Redis{name: "hello"}
	})

	dixglobal.Provide(func(l *log.Logger) map[string]*Redis {
		l.Println("init redis")
		return map[string]*Redis{
			"ns": {name: "hello1"},
		}
	})

	fmt.Println(dixglobal.Graph())

	dixglobal.Inject(func(r *Redis, l *log.Logger, rr map[string]*Redis) {
		l.Println("invoke redis")
		fmt.Println("invoke:", r.name)
		fmt.Println("invoke:", rr)
	})

	h := dixglobal.Inject(new(Handler))
	if h.Cli.name != "hello" {
		panic("inject error")
	}
	if h.Cli1["ns"].name != "hello1" {
		panic("inject error")
	}

	dixglobal.Inject(func(h Handler) {
		if h.Cli.name != "hello" {
			panic("inject error")
		}
		if h.Cli1["ns"].name != "hello1" {
			panic("inject error")
		}
	})
}
