package dix

import (
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"text/template"
)

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

func templates(s string, val interface{}) (string, error) {
	tpl, err := template.New("main").Delims("${", "}").Parse(s)
	if err != nil {
		return "", err
	}

	var buf = bytes.NewBuffer(nil)
	if err := tpl.Execute(buf, val); err != nil {
		return "", err
	}

	return buf.String(), nil
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

func recovery(fn func(err *Err)) {
	var err = recover()
	switch err.(type) {
	case nil:
		return
	case error:
		fn(&Err{Err: err.(error)})
	case string:
		fn(&Err{Msg: err.(string)})
	default:
		fn(&Err{Msg: fmt.Sprintf("%#v", err)})
	}
}
