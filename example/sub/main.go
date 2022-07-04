package main

import (
	"log"

	"github.com/pubgo/dix"
	"github.com/pubgo/funk"
)

func main() {
	defer funk.RecoverAndExit()
	dix.Provide(func() *log.Logger {
		return log.Default()
	})

	var sub = dix.SubDix()
	sub.Provider(func() *funk.Err {
		return &funk.Err{Msg: "ok"}
	})

	var err error
	funk.TryWith(&err, func() {
		dix.Inject(func(logger *log.Logger) {})
	})
	funk.Assert(err != nil, "inject failed")

	funk.TryWith(&err, func() {
		dix.Inject(func(err *funk.Err) {})
	})
	if err != nil {
		//xerr.WrapXErr(err).DebugPrint()
	}

	funk.Assert(err == nil, "inject error")

	err = nil
	funk.TryWith(&err, func() {
		sub.Inject(func(err *funk.Err) {})
	})
	funk.Assert(err != nil, "inject failed")

	funk.TryWith(&err, func() {
		sub.Inject(func(logger *log.Logger) {})
	})
	funk.Assert(err != nil, "inject failed")

}
