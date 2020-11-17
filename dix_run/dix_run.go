package dix_run

import "github.com/pubgo/dix"

type StartCtx struct{ dix.Model }
type BeforeStartCtx struct{ dix.Model }
type AfterStartCtx struct{ dix.Model }
type StopCtx struct{ dix.Model }
type BeforeStopCtx struct{ dix.Model }
type AfterStopCtx struct{ dix.Model }

func Start() error                                       { return dix.Dix(StartCtx{}) }
func WithStart(fn func(ctx *StartCtx)) error             { return dix.Dix(fn) }
func BeforeStart() error                                 { return dix.Dix(BeforeStartCtx{}) }
func WithBeforeStart(fn func(ctx *BeforeStartCtx)) error { return dix.Dix(fn) }
func AfterStart() error                                  { return dix.Dix(AfterStartCtx{}) }
func WithAfterStart(fn func(ctx *AfterStartCtx)) error   { return dix.Dix(fn) }
func Stop() error                                        { return dix.Dix(StopCtx{}) }
func WithStop(fn func(ctx *StopCtx)) error               { return dix.Dix(fn) }
func BeforeStop() error                                  { return dix.Dix(BeforeStopCtx{}) }
func WithBeforeStop(fn func(ctx *BeforeStopCtx)) error   { return dix.Dix(fn) }
func AfterStop() error                                   { return dix.Dix(AfterStopCtx{}) }
func WithAfterStop(fn func(ctx *AfterStopCtx)) error     { return dix.Dix(fn) }
