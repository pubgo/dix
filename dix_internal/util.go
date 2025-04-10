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

func handleOutput(outType outputType, providerOutTyp reflect.Value) map[outputType]map[group][]value {
	rr := make(map[outputType]map[group][]value)
	if !providerOutTyp.IsValid() || providerOutTyp.IsZero() {
		return rr
	}

	switch providerOutTyp.Kind() {
	case reflect.Map:
		outType = providerOutTyp.Type().Elem()
		isList := outType.Kind() == reflect.Slice
		if isList {
			outType = outType.Elem()
		}

		if rr[outType] == nil {
			rr[outType] = make(map[group][]value)
		}

		for _, k := range providerOutTyp.MapKeys() {
			mapK := strings.TrimSpace(k.String())
			if mapK == "" {
				mapK = defaultKey
			}

			val := providerOutTyp.MapIndex(k)
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
		outType = providerOutTyp.Type().Elem()
		if rr[outType] == nil {
			rr[outType] = make(map[group][]value)
		}

		for i := 0; i < providerOutTyp.Len(); i++ {
			val := providerOutTyp.Index(i)
			if !val.IsValid() || val.IsNil() {
				continue
			}

			rr[outType][defaultKey] = append(rr[outType][defaultKey], val)
		}
	case reflect.Struct:
		for i := 0; i < providerOutTyp.NumField(); i++ {
			for typ, vv := range handleOutput(providerOutTyp.Field(i).Type(), providerOutTyp.Field(i)) {
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

		if providerOutTyp.IsValid() && !providerOutTyp.IsNil() {
			rr[outType][defaultKey] = []value{providerOutTyp}
		}
	}
	return rr
}

func detectCycle(graph map[reflect.Type]map[reflect.Type]bool) []reflect.Type {
	visited := make(map[reflect.Type]bool)
	recursionStack := make(map[reflect.Type]bool)

	var dfs func(reflect.Type, []reflect.Type) []reflect.Type
	dfs = func(t reflect.Type, path []reflect.Type) []reflect.Type {
		if recursionStack[t] {
			return append([]reflect.Type(nil), path...)
		}

		if visited[t] {
			return nil
		}

		visited[t] = true
		recursionStack[t] = true
		defer delete(recursionStack, t)

		for dep := range graph[t] {
			cycle := dfs(dep, append(path, dep))
			if len(cycle) > 0 {
				return cycle
			}
		}
		return nil
	}

	for t := range graph {
		if visited[t] {
			continue
		}

		cycle := dfs(t, []reflect.Type{t})
		if len(cycle) > 0 {
			return cycle
		}
	}
	return nil
}
