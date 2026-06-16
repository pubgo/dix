// Typical wiring pattern: logger + service + handler struct injection.
//
// Run:
//
//	cd example/handler && go run .
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pubgo/dix/v2"
)

type Redis struct {
	Addr string
}

type Handler struct {
	Logger *log.Logger
	Redis  *Redis
	All    map[string]*Redis
}

func main() {
	di := dix.New()

	dix.Provide(di, func() *log.Logger {
		return log.New(os.Stderr, "app: ", log.LstdFlags)
	})

	dix.Provide(di, func(l *log.Logger) *Redis {
		l.Println("init default redis")
		return &Redis{Addr: "127.0.0.1:6379"}
	})

	dix.Provide(di, func(l *log.Logger) map[string]*Redis {
		l.Println("init namespaced redis map")
		return map[string]*Redis{
			"cache": {Addr: "127.0.0.1:6380"},
		}
	})

	// Function injection
	dix.Inject(di, func(r *Redis, l *log.Logger) {
		l.Println("invoke default redis:", r.Addr)
	})

	// Struct injection
	h := &Handler{}
	dix.Inject(di, h)

	fmt.Println("handler.Redis:", h.Redis.Addr)
	fmt.Println("handler.All[cache]:", h.All["cache"].Addr)
}
