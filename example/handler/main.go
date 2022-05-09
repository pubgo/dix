package main

import (
	"fmt"

	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
)

type Redis struct {
	Name string
}

type Handler struct {
	Name string
	// 如果是结构体，且tag为dix，那么，会检查结构体内部有指针或者接口属性，然后进行对象注入
	Cli  *Redis `dix:""`
	Cli1 *Redis `dix:"${.Name}"`
}

func main() {
	xerror.Exit(dix.Provider(&Redis{Name: "hello"}))
	xerror.Exit(dix.ProviderNs("ns", &Redis{Name: "hello1"}))

	fmt.Println(dix.Json())

	var h = Handler{Name: "ns"}
	xerror.Exit(dix.Inject(&h))
	fmt.Println(h.Cli.Name)  // hello
	fmt.Println(h.Cli1.Name) // hello
}
