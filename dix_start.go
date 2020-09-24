package dix

import "time"

type StartCtx struct {
	data int64
}

type StopCtx struct {
	data int64
}

func (x *dix) start() error {
	return x.dix(&StartCtx{time.Now().UnixNano()})
}

func (x *dix) stop() error {
	return x.dix(&StopCtx{time.Now().UnixNano()})
}
