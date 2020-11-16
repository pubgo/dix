package dix

import "github.com/pubgo/xerror"

var defaultDix = New()

// Init ...
func Init(opts ...Option) error { return defaultDix.Init(opts...) }

// Dix ...
func Dix(data ...interface{}) error { return defaultDix.Dix(data...) }

// Go the dix must be ok
func Go(data ...interface{}) { xerror.Exit(defaultDix.Dix(data...)) }

// Graph dix graph
func Graph() string { return defaultDix.graph() }

func Start() error                                       { return defaultDix.start() }
func WithStart(fn func(ctx *StartCtx)) error             { return defaultDix.withStart(fn) }
func BeforeStart() error                                 { return defaultDix.beforeStart() }
func WithBeforeStart(fn func(ctx *BeforeStartCtx)) error { return defaultDix.withBeforeStart(fn) }
func AfterStart() error                                  { return defaultDix.afterStart() }
func WithAfterStart(fn func(ctx *AfterStartCtx)) error   { return defaultDix.withAfterStart(fn) }
func Stop() error                                        { return defaultDix.stop() }
func WithStop(fn func(ctx *StopCtx)) error               { return defaultDix.withStop(fn) }
func BeforeStop() error                                  { return defaultDix.beforeStop() }
func WithBeforeStop(fn func(ctx *BeforeStopCtx)) error   { return defaultDix.withBeforeStop(fn) }
func AfterStop() error                                   { return defaultDix.afterStop() }
func WithAfterStop(fn func(ctx *AfterStopCtx)) error     { return defaultDix.withAfterStop(fn) }
