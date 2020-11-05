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

func (t Model) Init() () {
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

type startCtx struct{ Model }
type beforeStartCtx struct{ Model }
type afterStartCtx struct{ Model }
type stopCtx struct{ Model }
type beforeStopCtx struct{ Model }
type afterStopCtx struct{ Model }

func (x *dix) start() error {
	return x.dix(startCtx{})
}

func (x *dix) withStart(fn func()) error {
	return x.dix(func(ctx *startCtx) { fn() })
}

func (x *dix) beforeStart() error {
	return x.dix(beforeStartCtx{})
}

func (x *dix) withBeforeStart(fn func()) error {
	return x.dix(func(ctx *beforeStartCtx) { fn() })
}

func (x *dix) afterStart() error {
	return x.dix(afterStartCtx{})
}

func (x *dix) withAfterStart(fn func()) error {
	return x.dix(func(ctx *afterStartCtx) { fn() })
}

func (x *dix) stop() error {
	return x.dix(stopCtx{})
}

func (x *dix) withStop(fn func()) error {
	return x.dix(func(ctx *stopCtx) { fn() })
}

func (x *dix) beforeStop() error {
	return x.dix(beforeStopCtx{})
}

func (x *dix) withBeforeStop(fn func()) error {
	return x.dix(func(ctx *beforeStopCtx) { fn() })
}

func (x *dix) afterStop() error {
	return x.dix(afterStopCtx{})
}

func (x *dix) withAfterStop(fn func()) error {
	return x.dix(func(ctx *afterStopCtx) { fn() })
}
