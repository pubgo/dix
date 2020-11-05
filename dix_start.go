package dix

import "time"

type startCtx struct {
	data time.Time
}

type beforeStartCtx struct {
	data time.Time
}

type afterStartCtx struct {
	data time.Time
}

type stopCtx struct {
	data time.Time
}

type beforeStopCtx struct {
	data time.Time
}

type afterStopCtx struct {
	data time.Time
}

func (x *dix) start() error {
	return x.dix(&startCtx{time.Now()})
}

func (x *dix) withStart(fn func()) error {
	return x.dix(func(ctx *startCtx) { fn() })
}

func (x *dix) beforeStart() error {
	return x.dix(&beforeStartCtx{time.Now()})
}

func (x *dix) withBeforeStart(fn func()) error {
	return x.dix(func(ctx *beforeStartCtx) { fn() })
}

func (x *dix) afterStart() error {
	return x.dix(&afterStartCtx{time.Now()})
}

func (x *dix) withAfterStart(fn func()) error {
	return x.dix(func(ctx *afterStartCtx) { fn() })
}

func (x *dix) stop() error {
	return x.dix(&stopCtx{time.Now()})
}

func (x *dix) withStop(fn func()) error {
	return x.dix(func(ctx *stopCtx) { fn() })
}

func (x *dix) beforeStop() error {
	return x.dix(&beforeStopCtx{time.Now()})
}

func (x *dix) withBeforeStop(fn func()) error {
	return x.dix(func(ctx *beforeStopCtx) { fn() })
}

func (x *dix) afterStop() error {
	return x.dix(&afterStopCtx{time.Now()})
}

func (x *dix) withAfterStop(fn func()) error {
	return x.dix(func(ctx *afterStopCtx) { fn() })
}
