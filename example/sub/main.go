package main

import (
	"log"

	"github.com/pubgo/funk"
	"github.com/pubgo/funk/assert"
	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/xerr"

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

	var err error
	funk.TryWith(&err, func() {
		dix.Inject(func(logger *log.Logger) {})
	})
	assert.If(err != nil, "inject failed")

	funk.TryWith(&err, func() {
		dix.Inject(func(err *xerr.Err) {})
	})
	if err != nil {
		//xerr.WrapXErr(err).DebugPrint()
	}

	assert.If(err == nil, "inject error")

	err = nil
	funk.TryWith(&err, func() {
		sub.Inject(func(err *xerr.Err) {})
	})
	assert.If(err != nil, "inject failed")

	funk.TryWith(&err, func() {
		sub.Inject(func(logger *log.Logger) {})
	})
	assert.If(err != nil, "inject failed")
}
