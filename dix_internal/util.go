package dix_internal

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

func detectCycle(graph map[reflect.Type]map[reflect.Type]bool) []reflect.Type {
	visited := make(map[reflect.Type]bool)
	recursionStack := make(map[reflect.Type]bool)

	var cycle []reflect.Type

	var dfs func(reflect.Type, []reflect.Type)
	dfs = func(t reflect.Type, path []reflect.Type) {
		if recursionStack[t] {
			cycle = append([]reflect.Type(nil), path...)
			return
		}
		if visited[t] {
			return
		}

		visited[t] = true
		recursionStack[t] = true
		defer delete(recursionStack, t)

		for dep := range graph[t] {
			dfs(dep, append(path, dep))
			if len(cycle) > 0 {
				return
			}
		}
	}

	for t := range graph {
		if !visited[t] {
			dfs(t, []reflect.Type{t})
		}
	}
	return cycle
}
