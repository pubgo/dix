package main

import (
	"github.com/kr/pretty"
	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/recovery"
)

type AP struct {
	BI
}

type A struct {
	AP
}

type A1 struct {
}

type AR struct {
	*A
}

func newA(p AP) AR {
	return AR{
		A: &A{p},
	}
}

type BP struct {
}

type B struct {
	BP
}

type BI interface{}

type BR struct {
	BI
}

func newB(p BP) BR {
	return BR{}
}

type params struct {
	*A
}

func Main() {
	defer recovery.Exit()

	dixglobal.Provide(newA)
	dixglobal.Provide(newB)

	pretty.Println(dixglobal.Inject(&params{}))
}

func main() {
	Main()
}
