package dix_trace

import (
	"expvar"

	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
)

type TraceCtx struct{ dix.Model }

func Trace() error              { return xerror.Wrap(dix.Dix(TraceCtx{})) }
func With(fn func(_ *TraceCtx)) { xerror.Next().Panic(dix.Dix(fn)) }

func init() {
	With(func(_ *TraceCtx) { expvar.NewString("dix").Set(dix.Graph()) })
}
