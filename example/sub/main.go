package main

import (
	"log"

	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/xerr"
	"github.com/pubgo/funk/xtry"

	"github.com/pubgo/dix"
)

func main() {
	defer recovery.Exit()
	dix.Provide(func() *log.Logger {
		return log.Default()
	})

	var sub = dix.SubDix()
	sub.Provider(func() *xerr.Err {
		return &xerr.Err{Msg: "ok"}
	})

	xtry.Try(func() {
		dix.Inject(func(logger *log.Logger) {})
		dix.Inject(func(err *xerr.Err) {})
		sub.Inject(func(err *xerr.Err) {})
		sub.Inject(func(logger *log.Logger) {})
	}).Expect("inject failed")
}
