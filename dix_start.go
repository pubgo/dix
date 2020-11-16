package dix

import (
	"reflect"
	"sync/atomic"
	"time"
	"unsafe"
)

type Model struct {
	data unsafe.Pointer
}

func (t Model) Init() {
	var data = time.Now()
	atomic.StorePointer(&t.data, unsafe.Pointer(&data))
}

type dixData interface {
	Init()
}

// checkDixDataType
// 检查是否实现dixData
func checkDixDataType(data dixData) interface{} {
	dt := reflect.New(unWrapType(reflect.TypeOf(data)))
	dt.MethodByName("Init").Call([]reflect.Value{})
	return dt.Interface()
}

type StartCtx struct{ Model }
type BeforeStartCtx struct{ Model }
type AfterStartCtx struct{ Model }
type StopCtx struct{ Model }
type BeforeStopCtx struct{ Model }
type AfterStopCtx struct{ Model }

func (x *dix) start() error {
	return x.dix(StartCtx{})
}

func (x *dix) withStart(fn func(ctx *StartCtx)) error {
	return x.dix(fn)
}

func (x *dix) beforeStart() error {
	return x.dix(BeforeStartCtx{})
}

func (x *dix) withBeforeStart(fn func(ctx *BeforeStartCtx)) error {
	return x.dix(fn)
}

func (x *dix) afterStart() error {
	return x.dix(AfterStartCtx{})
}

func (x *dix) withAfterStart(fn func(ctx *AfterStartCtx)) error {
	return x.dix(fn)
}

func (x *dix) stop() error {
	return x.dix(StopCtx{})
}

func (x *dix) withStop(fn func(ctx *StopCtx)) error {
	return x.dix(fn)
}

func (x *dix) beforeStop() error {
	return x.dix(BeforeStopCtx{})
}

func (x *dix) withBeforeStop(fn func(ctx *BeforeStopCtx)) error {
	return x.dix(fn)
}

func (x *dix) afterStop() error {
	return x.dix(AfterStopCtx{})
}

func (x *dix) withAfterStop(fn func(ctx *AfterStopCtx)) error {
	return x.dix(fn)
}
