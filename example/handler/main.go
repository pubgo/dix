package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pubgo/dix"

	"github.com/pubgo/dix/internal/assert"
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
			"default": {Name: "hello"},
		}
	})

	dix.Register(func(l *log.Logger) map[string]*Redis {
		l.Println("init redis")
		return map[string]*Redis{
			"ns": {Name: "hello1"},
		}
	})

	dix.Register(func(r *Redis, l *log.Logger, rr map[string]*Redis) {
		l.Println("invoke redis")
		fmt.Println("invoke:", r.Name)
		fmt.Println("invoke:", rr)
	})

	var h = Handler{Name: "ns"}
	dix.Inject(&h)
	assert.Msg(h.Cli.Name != "hello", "inject error")
	assert.Msg(h.Cli1.Name != "hello1", "inject error")
	dix.Invoke()
	fmt.Println(dix.Graph())
}
