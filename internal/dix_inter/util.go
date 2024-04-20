package dix_inter

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pubgo/funk/errors"
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
	val := reflect.MakeSlice(reflect.SliceOf(typ), 0, 0)
	return reflect.Append(val, data...)
}

func makeMap(typ reflect.Type, data map[string][]reflect.Value, valueList bool) reflect.Value {
	if valueList {
		typ = reflect.SliceOf(typ)
	}

	mapVal := reflect.MakeMap(reflect.MapOf(reflect.TypeOf(""), typ))
	for index, values := range data {
		// 最后一个值作为默认值
		val := values[len(values)-1]
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
	rr := make(map[outputType]map[group][]value)
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
			mapK := strings.TrimSpace(k.String())
			if mapK == "" {
				mapK = defaultKey
			}

			val := out.MapIndex(k)
			if !val.IsValid() || val.IsNil() {
				continue
			}

			if isList {
				for i := 0; i < val.Len(); i++ {
					vv := val.Index(i)
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
			val := out.Index(i)
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

// get all provider input type, include struct inner type
func getAllInputType(inTye reflect.Type) []*inType {
	var input []*inType
	switch inTye.Kind() {
	case reflect.Interface, reflect.Ptr, reflect.Func:
		input = append(input, &inType{typ: inTye})
	case reflect.Struct:
		for j := 0; j < inTye.NumField(); j++ {
			input = append(input, getAllInputType(inTye.Field(j).Type)...)
		}
	case reflect.Map:
		tt := &inType{typ: inTye.Elem(), isMap: true, isList: inTye.Elem().Kind() == reflect.Slice}

		// map[string][]Object
		if tt.isList {
			tt.typ = tt.typ.Elem()
		}
		input = append(input, tt)
	case reflect.Slice:
		input = append(input, &inType{typ: inTye.Elem(), isList: true})
	default:
		panic(&errors.Err{
			Msg:    "incorrect input type",
			Detail: fmt.Sprintf("inTyp=%s kind=%s", inTye, inTye.Kind()),
			Tags:   []errors.Tag{},
		})
	}
	return input
}

// getProvideInput get provider input type without strcut inner type
func getProvideInput(typ reflect.Type) []*inType {
	var input []*inType
	switch inTye := typ; inTye.Kind() {
	case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct:
		input = append(input, &inType{typ: inTye})
	case reflect.Map:
		tt := &inType{typ: inTye.Elem(), isMap: true, isList: inTye.Elem().Kind() == reflect.Slice}
		if tt.isList {
			tt.typ = tt.typ.Elem()
		}
		input = append(input, tt)
	case reflect.Slice:
		input = append(input, &inType{typ: inTye.Elem(), isList: true})
	default:
		panic(&errors.Err{
			Msg:    "incorrect input type",
			Detail: fmt.Sprintf("inTyp=%s kind=%s", inTye, inTye.Kind()),
		})
	}
	return input
}
