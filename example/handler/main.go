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
	Cli Client `dix:""`
}

func main() {
	xerror.Exit(dix.Provider(&Redis{Name: "hello"}))

	var h Handler
	xerror.Exit(dix.Inject(&h))
	fmt.Println(h.Cli.Name)
}
