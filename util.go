package dix

import (
	"bytes"
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

func makeList(data []reflect.Value) reflect.Value {
	var kt = data[0].Type()
	var val = reflect.MakeSlice(reflect.SliceOf(kt), 0, 0)
	return reflect.Append(val, data...)
}

func makeMap(data map[string][]reflect.Value) reflect.Value {
	var kt reflect.Type
	for k := range data {
		kt = data[k][0].Type()
		break
	}

	var mapVal = reflect.MakeMap(reflect.MapOf(reflect.TypeOf(""), kt))
	for k, v := range data {
		mapVal.SetMapIndex(reflect.ValueOf(k), v[len(v)-1])
	}
	return mapVal
}
