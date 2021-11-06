# dix
> dix是一个参考了dig设计的依赖注入工具

> dix和dig的主要区别在于dix能够完成更加复杂的依赖注入管理和namespace依赖隔离，同时dix也能动态的绑定对象到依赖函数, 对象变更通知等

> dix使用非常方便, 通过Dix可以进行对象的发布更新和监听绑定等

## 使用

```go
package main

import (
	"github.com/pubgo/dix"
)

var _ error = (*Data)(nil)

type Data struct {
	hello string
}

func (d Data) Error() string {
	return d.hello
}

func init() {
    // 对象类型和namespace是全局唯一的key

	// data 参数可以是ptr，interface，struct，map和func
	dix.Provider(&Data{hello: "hello"})
	dix.ProviderNs("ns", &Data{hello: "hello"}) // 设置namespace, 默认为default

	dix.Provider(error(&Data{hello: "hello"}))
	dix.Provider(struct {
		D *Data `dix:""`
	}{D: &Data{hello: "hello"}})
	dix.Provider(map[string]interface{}{"hello": &Data{hello: "hello"}})

	// 如果provider为func，那么func的参数可以为ptr，interface，struct
	dix.Provider(func(data *Data) {})
	dix.Provider(func(data Data) {})
	dix.Provider(func(data error) {})

	// 依赖注入,data要注入的对象, 必须为ptr类型，ns为namespace, 默认为default
	type A struct {
		D *Data `dix:""` // 属性必须是可导出的，同时tag必须为dix
	}

	var a A        // 注入对象到结构体
	dix.Inject(&a) // 注入Data对象到a的D中

	var d *Data    // 注入对象到指针
	dix.Inject(&d) // 注入Data对象到d中

	var e error    // 注入对象到interface
	dix.Inject(&e) // 注入Data对象到e中

	// 查看依赖关系图
	dix.Graph()
}
```

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

type m11 struct {
}

func (t *m11) Hello1() {
	fmt.Println("Hello1")
}

type M1 interface {
	Hello1()
}

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
		L *log.Logger `dix:""`
		H Hello       `dix:"test"`
	}

	xerror.Exit(dix.Dix(func(l ll) {
		fmt.Println(l)
		l.L.Print("You've been invoked2")
		l.H.Hello()
	}))
}

type M2 interface {
	A()
}

type m22 struct {
	l *log.Logger
}

func (t *m22) A() {
	t.l.Println("AAA")
}

func NewM22(l *log.Logger) M2 {
	return &m22{l: l}
}

func init() {
	xerror.Exit(dix.Dix(func(l *log.Logger) (map[string]M2, error) {
		l.Println("m22 start")
		return map[string]M2{"hello": NewM22(l)}, nil
	}))

	type nss struct {
		M2 M2 `dix:"hello"`
	}

	xerror.Exit(dix.Dix(func(l nss) {
		log.Println("nss start")
		l.M2.A()
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
		//= &MM{Abc: "hello MM"}

		type nn struct {
			Struct1 Hello `dix:""`
			MM      string
		}
		var struct1 = &nn{MM: "ssss"}
		xerror.Panic(dix.Invoke(struct1))
		struct1.Struct1.Hello()
		fmt.Println(struct1.MM)

		xerror.Panic(dix.Dix(&m11{}))
		fmt.Println(dix.Graph())

		var mmm struct {
			M1 M1 `dix:""`
		}
		xerror.Panic(dix.Invoke(&mmm))
		mmm.M1.Hello1()
		i++
	}
}
```