# dix
> dix是一个参考了dig设计的依赖注入工具

> dix和dig的主要区别在于dix能够完成更加复杂的依赖注入管理和更加优秀的依靠namespace的依赖隔离，同时dix也能动态的绑定对象到依赖函数, 对象变更通知等

> dix 使用非常方便, 通过Dix可以进行对象的发布更新和监听绑定等

## [更新日志](./CHANGELOG.md)

## example

```go
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

func init() {
	dix.Go(func(h *testHello) {
		fmt.Println("h *testHello")
	})

	xerror.Exit(dix.Dix(func(h Hello) {
		h.Hello()
	}))

	xerror.Exit(dix.Dix(func(cfg *Config) (*log.Logger, error) {
		fmt.Println("cfg *Config")
		return log.New(os.Stdout, cfg.Prefix, log.Llongfile), nil
	}))

	xerror.Exit(dix.Dix(func(l *log.Logger) {
		l.Print("You've been invoked")
	}))

	type ll struct {
		L *log.Logger
		H Hello
	}
	xerror.Exit(dix.Dix(func(l ll) {
		l.L.Print("You've been invoked")
		l.H.Hello()
	}))
}

func main() {
	i := 0
	for {
		var cfg Config
		xerror.Exit(json.Unmarshal([]byte(fmt.Sprintf(`{"prefix": "[foo%d] "}`, i)), &cfg))
		dix.Go(&cfg)

		fmt.Println(dix.Graph())
		fmt.Print("==================================================================================\n")
		time.Sleep(time.Second)
		xerror.Exit(dix.Dix(&testHello{i: i}))
		fmt.Println(dix.Graph())
		time.Sleep(time.Second)
		i++
	}
}
```