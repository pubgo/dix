package dix

import (
	"reflect"
)

type inType struct {
	typ   reflect.Type
	isMap bool
}

type outType struct {
	typ   reflect.Type
	isMap bool
}

type node struct {
	fn     reflect.Value
	input  []*inType
	output *outType
}

func (n node) call(in []reflect.Value) []reflect.Value {
	defer recovery(func(err *Err) {
		logs.Println("provider call failed")
		logs.Println("provider is", callerWithFunc(n.fn))
		logs.Printf("provider input is %v\n", in)
		panic(err)
	})

	return n.fn.Call(in)
}
