package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pubgo/dix"
)

type Redis struct {
	Name string
}

type Handler struct {
	Name string
	// 如果是结构体，且tag为dix，那么，会检查结构体内部有指针或者接口属性，然后进行对象注入
	Cli  *Redis `inject:""`
	Cli1 *Redis `inject:"${.Name}"`
}

func main() {
	dix.Register(func() *log.Logger {
		return log.New(os.Stderr, "", log.LstdFlags|log.Llongfile)
	})

	dix.Register(func(l *log.Logger) map[string]*Redis {
		l.Println("init redis")
		return map[string]*Redis{
			"default": &Redis{Name: "hello"},
			"ns":      &Redis{Name: "hello1"},
		}
	})

	dix.Register(func(r *Redis, l *log.Logger, rr map[string]*Redis) {
		l.Println("invoke redis")
		fmt.Println("invoke:", r.Name)
		fmt.Println("invoke:", rr)
	})

	var h = Handler{Name: "ns"}
	dix.Inject(&h)
	fmt.Println(h.Cli.Name)  // hello
	fmt.Println(h.Cli1.Name) // hello
	dix.Invoke()
}
