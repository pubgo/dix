package main

import (
	"fmt"

	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
)

type Redis struct {
	Name string
}

type Client struct {
	*Redis `dix:""`
}

type Handler struct {
	// 如果是结构体，且tag为dix，那么，会检查结构体内部有指针或者接口属性，然后进行对象注入
	Cli Client `dix:""`
}

func main() {
	xerror.Exit(dix.Provider(&Redis{Name: "hello"}))

	var h Handler
	xerror.Exit(dix.Inject(&h))
	fmt.Println(h.Cli.Name) // hello
}
