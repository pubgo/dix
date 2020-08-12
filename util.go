package dix

import (
	"fmt"
	"io"
	"reflect"
)

var _errType = reflect.TypeOf((*error)(nil)).Elem()

func isError(t reflect.Type) bool {
	return t.Implements(_errType)
}

func unWrapType(tye reflect.Type) reflect.Type {
	for isElem(tye) {
		tye = tye.Elem()
	}
	return tye
}

func isElem(tye reflect.Type) bool {
	switch tye.Kind() {
	case reflect.Chan, reflect.Map, reflect.Ptr, reflect.Array, reflect.Slice:
		return true
	default:
		return false
	}
}

func fPrintln(w io.Writer, a ...interface{}) {
	_, _ = fmt.Fprintln(w, a...)
}
