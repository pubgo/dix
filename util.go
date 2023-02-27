package dix

import (
	"reflect"
)

func checkType(p reflect.Kind) bool {
	switch p {
	case reflect.Interface, reflect.Ptr, reflect.Func:
		return true
	default:
		return false
	}
}

func makeList(typ reflect.Type, data []reflect.Value) reflect.Value {
	var val = reflect.MakeSlice(reflect.SliceOf(typ), 0, 0)
	return reflect.Append(val, data...)
}

func makeMap(typ reflect.Type, data map[string][]reflect.Value) reflect.Value {
	var mapVal = reflect.MakeMap(reflect.MapOf(reflect.TypeOf(""), typ))
	for k, v := range data {
		// 最后一个值作为默认值
		mapVal.SetMapIndex(reflect.ValueOf(k), v[len(v)-1])
	}
	return mapVal
}
