package dix_trace

import (
	"expvar"

	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
)

type TraceCtx struct{ dix.Model }
type Var = expvar.Var

func (t TraceCtx) Publish(name string, v expvar.Var) {
	expvar.Publish(name, v)
}

func (t TraceCtx) String(name string, data string) {
	expvar.NewString(name).Set(data)
}

func (t TraceCtx) Func(name string, data func() interface{}) {
	expvar.Publish(name, expvar.Func(data))
}

func (t TraceCtx) Float(name string, data float64) {
	expvar.NewFloat(name).Set(data)
}

func (t TraceCtx) Int(name string, data int64) {
	expvar.NewInt(name).Set(data)
}

func Trace() error                { return xerror.Wrap(dix.Dix(TraceCtx{})) }
func With(fn func(ctx *TraceCtx)) { xerror.Next().Panic(dix.Dix(fn)) }

func init() {
	With(func(ctx *TraceCtx) { ctx.String("dix", dix.Graph()) })
}
