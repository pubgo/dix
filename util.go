package dix

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/pubgo/xerror"
)

var errType = reflect.TypeOf((*error)(nil)).Elem()

func isError(t reflect.Type) bool {
	return t.Implements(errType)
}

func isElem(tye reflect.Type) bool {
	switch tye.Kind() {
	case reflect.Ptr:
		return true
	default:
		return false
	}
}

func getIndirectType(tye reflect.Type) reflect.Type {
	for isElem(tye) {
		tye = tye.Elem()
	}
	return tye
}

func fPrintln(w io.Writer, a ...interface{}) {
	_, _ = fmt.Fprintln(w, a...)
}

func callerWithFunc(fn reflect.Value) string {
	var _e = runtime.FuncForPC(fn.Pointer())
	var file, line = _e.FileLine(fn.Pointer())

	var buf = &strings.Builder{}
	defer buf.Reset()

	files := strings.Split(file, "/")
	if len(files) > 2 {
		file = strings.Join(files[len(files)-2:], "/")
	}

	buf.WriteString(file)
	buf.WriteString(":")
	buf.WriteString(strconv.Itoa(line))
	buf.WriteString(" ")

	buf.WriteString(fn.String())
	return buf.String()
}

func equal(x, y []reflect.Value) bool {
	if len(x) != len(y) {
		return false
	}

	for i := range x {
		if x[i].IsNil() || y[i].IsNil() {
			return false
		}

		if x[i] != y[i] {
			return false
		}
	}
	return true
}

func templates(s string, val interface{}) string {
	tpl, err := template.New("main").Delims("${", "}").Parse(s)
	xerror.Panic(err)
	var buf = bytes.NewBuffer(nil)
	xerror.Panic(tpl.Execute(buf, val))
	return buf.String()
}

func makeMap(data map[string]reflect.Value) reflect.Value {
	if len(data) == 0 {
		return reflect.Value{}
	}

	var kt reflect.Type
	for k := range data {
		kt = data[k].Type()
		break
	}

	var mapVal = reflect.MakeMap(reflect.MapOf(reflect.TypeOf(""), kt))
	for k, v := range data {
		mapVal.SetMapIndex(reflect.ValueOf(k), v)
	}
	return mapVal
}
