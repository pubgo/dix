package dix_inter

import (
	"fmt"
	"reflect"
	"strings"
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

func reflectValueToString(values []reflect.Value) []string {
	var data []string
	for i := range values {
		data = append(data, fmt.Sprintf("%#v", values[i].Interface()))
	}
	return data
}

func handleOutput(outType outputType, out reflect.Value) map[outputType]map[group][]value {
	var rr = make(map[outputType]map[group][]value)
	if !out.IsValid() || out.IsZero() {
		return rr
	}

	switch out.Kind() {
	case reflect.Map:
		outType = out.Type().Elem()
		isList := outType.Kind() == reflect.Slice
		if isList {
			outType = outType.Elem()
		}

		if rr[outType] == nil {
			rr[outType] = make(map[group][]value)
		}

		for _, k := range out.MapKeys() {
			var mapK = strings.TrimSpace(k.String())
			if mapK == "" {
				mapK = defaultKey
			}

			var val = out.MapIndex(k)
			if !val.IsValid() || val.IsNil() {
				continue
			}

			if isList {
				for i := 0; i < val.Len(); i++ {
					var vv = val.Index(i)
					if !vv.IsValid() || vv.IsNil() {
						continue
					}

					rr[outType][mapK] = append(rr[outType][mapK], vv)
				}
			} else {
				rr[outType][mapK] = append(rr[outType][mapK], val)
			}
		}
	case reflect.Slice:
		outType = out.Type().Elem()
		if rr[outType] == nil {
			rr[outType] = make(map[group][]value)
		}

		for i := 0; i < out.Len(); i++ {
			var val = out.Index(i)
			if !val.IsValid() || val.IsNil() {
				continue
			}

			rr[outType][defaultKey] = append(rr[outType][defaultKey], val)
		}
	case reflect.Struct:
		for i := 0; i < out.NumField(); i++ {
			for typ, vv := range handleOutput(out.Field(i).Type(), out.Field(i)) {
				if rr[typ] == nil {
					rr[typ] = vv
				} else {
					for g, v := range vv {
						rr[typ][g] = append(rr[typ][g], v...)
					}
				}
			}
		}
	default:
		if rr[outType] == nil {
			rr[outType] = make(map[group][]value)
		}

		if out.IsValid() && !out.IsNil() {
			rr[outType][defaultKey] = []value{out}
		}
	}
	return rr
}
