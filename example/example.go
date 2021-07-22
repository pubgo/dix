package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
)

type Hello interface {
	Hello()
}

type testHello struct {
	i int
}

func (t testHello) Hello() {
	fmt.Println("config testHello")
}

type Config struct {
	Prefix string
}

func (Config) Hello() {
	fmt.Println("Hello Config")
}

type MM struct {
	Cfg *Config `dix:"test"`
	Abc string
}

func (MM) Hello() {
	fmt.Println("Hello MM")
}

func init() {
	xerror.Panic(dix.Dix(func(h *testHello) {
		fmt.Println("h *testHello")
	}))

	xerror.Exit(dix.Dix(func(h Hello) {
		h.Hello()
	}))

	xerror.Exit(dix.Dix(func(cfg MM) (*log.Logger, error) {
		fmt.Println("cfg *Config")
		fmt.Println(cfg.Cfg)
		return log.New(os.Stdout, cfg.Cfg.Prefix, log.Llongfile), nil
	}))

	xerror.Exit(dix.Dix(func(l *log.Logger) {
		fmt.Println(l)
		l.Print("You've been invoked1")
	}))

	type ll struct {
		L *log.Logger
		H Hello `dix:"test"`
	}

	xerror.Exit(dix.Dix(func(l ll) {
		fmt.Println(l)
		l.L.Print("You've been invoked2")
		l.H.Hello()
	}))
}

func main() {
	defer xerror.RespExit()

	i := 0
	for {
		var cfg Config
		xerror.Exit(json.Unmarshal([]byte(fmt.Sprintf(`{"prefix": "[foo%d] "}`, i)), &cfg))
		xerror.Panic(dix.Dix(map[string]*Config{"test": &cfg}))
		fmt.Printf("cfg: %#v\n", cfg)

		fmt.Println(dix.Graph())
		fmt.Print("==================================================================================\n")
		time.Sleep(time.Second)
		xerror.Exit(dix.Dix(&testHello{i: i}))
		fmt.Println(dix.Graph())
		//time.Sleep(time.Second)
		var log1 *log.Logger
		xerror.Panic(dix.Invoke(&log1))
		fmt.Printf("log1: %#v\n", log1)

		var cfg1 *Config
		xerror.Panic(dix.Invoke(&cfg1, "test"))
		fmt.Printf("cfg1: %#v\n", cfg1)

		// 接口类型
		var struct1 Hello = &MM{Abc: "hello MM"}
		xerror.Panic(dix.Invoke(struct1))
		fmt.Printf("struct1: %#v\n", struct1)
		i++
	}
}
