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
