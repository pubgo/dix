package dix_inter

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

func makeMap(typ reflect.Type, data map[string][]reflect.Value, valueList bool) reflect.Value {
	if valueList {
		typ = reflect.SliceOf(typ)
	}

	var mapVal = reflect.MakeMap(reflect.MapOf(reflect.TypeOf(""), typ))
	for index, values := range data {
		// 最后一个值作为默认值
		var val = values[len(values)-1]
		if valueList {
			val = reflect.MakeSlice(typ, 0, len(values))
			val = reflect.Append(val, values...)
		}
		mapVal.SetMapIndex(reflect.ValueOf(index), val)
	}
	return mapVal
}
