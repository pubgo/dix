package dix_trace

import (
	"expvar"

	"github.com/pubgo/dix"
	"github.com/pubgo/dix/dix_envs"
	"github.com/pubgo/xerror"
)

type Ctx struct{ dix.Model }
type Var = expvar.Var

func (t Ctx) Publish(name string, v expvar.Var)         { expvar.Publish(name, v) }
func (t Ctx) String(name string, data string)           { expvar.NewString(name).Set(data) }
func (t Ctx) Func(name string, data func() interface{}) { expvar.Publish(name, expvar.Func(data)) }
func (t Ctx) Float(name string, data float64)           { expvar.NewFloat(name).Set(data) }
func (t Ctx) Int(name string, data int64)               { expvar.NewInt(name).Set(data) }

func Trigger() error {
	if !dix_envs.Enabled() {
		return nil
	}

	return xerror.Wrap(dix.Dix(Ctx{}))
}
func With(fn func(ctx *Ctx)) { xerror.Next().Panic(dix.Dix(fn)) }

func init() {
	With(func(ctx *Ctx) { ctx.String("dix", dix.Graph()) })
}
