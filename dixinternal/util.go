package dixinternal

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
)

var tracePkgPathCache sync.Map

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
		// The last value as the default value
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

	// Deterministic traversal: iterate start nodes and neighbors in
	// sorted type-name order so the same graph always reports the same
	// cycle path (starting at its lexicographically smallest member),
	// regardless of Go's randomized map iteration order.
	sortTypes := func(types []reflect.Type) []reflect.Type {
		slices.SortFunc(types, func(a, b reflect.Type) int {
			return strings.Compare(a.String(), b.String())
		})
		return types
	}

	var dfs func(reflect.Type, map[reflect.Type]bool, []reflect.Type) []reflect.Type
	dfs = func(t reflect.Type, recursionStack map[reflect.Type]bool, path []reflect.Type) []reflect.Type {
		if recursionStack[t] {
			return trimCyclePath(slices.Clone(path))
		}

		if visited[t] {
			return nil
		}

		visited[t] = true
		recursionStack[t] = true
		defer delete(recursionStack, t)

		deps := make([]reflect.Type, 0, len(graph[t]))
		for dep := range graph[t] {
			deps = append(deps, dep)
		}
		for _, dep := range sortTypes(deps) {
			cycle := dfs(dep, recursionStack, append(slices.Clone(path), dep))
			if len(cycle) > 0 {
				return cycle
			}
		}
		return nil
	}

	starts := make([]reflect.Type, 0, len(graph))
	for t := range graph {
		starts = append(starts, t)
	}
	for _, t := range sortTypes(starts) {
		if visited[t] {
			continue
		}

		cycle := dfs(t, make(map[reflect.Type]bool), []reflect.Type{t})
		if len(cycle) > 0 {
			return cycle
		}
	}
	return nil
}

// trimCyclePath trims the cycle-external prefix from a DFS path so the reported
// cycle starts at the repeated node: X -> A -> B -> A is reported as A -> B -> A.
func trimCyclePath(path []reflect.Type) []reflect.Type {
	if len(path) < 2 {
		return path
	}
	last := path[len(path)-1]
	for i, t := range path[:len(path)-1] {
		if t == last {
			return path[i:]
		}
	}
	return path
}

func getProvideAllInputs(typ reflect.Type) []*providerInputType {
	var input []*providerInputType
	switch inTye := typ; inTye.Kind() {
	case reflect.Interface, reflect.Ptr, reflect.Func:
		input = append(input, &providerInputType{typ: inTye})
	case reflect.Struct:
		for j := 0; j < inTye.NumField(); j++ {
			if !inTye.Field(j).IsExported() {
				continue
			}

			inTyp := inTye.Field(j).Type
			if !isSupportedType(inTyp) {
				continue
			}

			input = append(input, getProvideAllInputs(inTyp)...)
		}
	case reflect.Map:
		tt := &providerInputType{typ: inTye.Elem(), isMap: true, isList: inTye.Elem().Kind() == reflect.Slice}
		if tt.isList {
			tt.typ = tt.typ.Elem()
		}
		input = append(input, tt)
	case reflect.Slice:
		input = append(input, &providerInputType{typ: inTye.Elem(), isList: true})
	default:
		logger.Warn("unsupported input type for dependency analysis", "type", inTye.String(), "kind", inTye.Kind().String())
	}
	return input
}

func buildDependencyGraph(providers map[outputType][]*providerFn) map[reflect.Type]map[reflect.Type]bool {
	graph := make(map[reflect.Type]map[reflect.Type]bool)
	// Pre-allocate map capacity to reduce rehash
	for outTyp := range providers {
		graph[outTyp] = make(map[reflect.Type]bool)
	}

	// Build dependency graph
	for outTyp, nodes := range providers {
		for _, providerNode := range nodes {
			for _, input := range providerNode.inputList {
				for _, provider := range getProvideAllInputs(input.typ) {
					graph[outTyp][provider.typ] = true
				}
			}
		}
	}
	return graph
}

// isSupportedType checks if the type is supported for dependency injection
func isSupportedType(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct:
		return true
	case reflect.Map, reflect.Slice:
		return isMapListSupportedType(typ.Elem())
	default:
		return false
	}
}

func isMapListSupportedType(p reflect.Type) bool {
	switch p.Kind() {
	case reflect.Interface, reflect.Ptr, reflect.Func:
		return true
	default:
		return false
	}
}

// GetFnName returns the name of the function represented by reflect.Value
func GetFnName(fn reflect.Value) string {
	if !fn.IsValid() || fn.IsNil() {
		return "nil"
	}
	pc := fn.Pointer()
	f := runtime.FuncForPC(pc)
	if f == nil {
		return "unknown"
	}
	return f.Name()
}

// GetFnTraceName returns function name for trace logs with best-effort full package path.
// For normal packages runtime already returns full import path.
// For `main.*` symbols, this attempts to rebuild a module-qualified path from source file location.
func GetFnTraceName(fn reflect.Value) string {
	name := GetFnName(fn)
	if name == "nil" || name == "unknown" {
		return name
	}

	if !strings.HasPrefix(name, "main.") {
		return name
	}

	pc := fn.Pointer()
	f := runtime.FuncForPC(pc)
	if f == nil {
		return name
	}

	file, _ := f.FileLine(pc)
	pkgPath := inferPkgPathFromFile(file)
	if pkgPath == "" {
		return name
	}

	suffix := strings.TrimPrefix(name, "main.")
	if suffix == name {
		return name
	}

	return pkgPath + "." + suffix
}

func inferPkgPathFromFile(file string) string {
	if strings.TrimSpace(file) == "" {
		return ""
	}

	fileDir := filepath.Clean(filepath.Dir(file))
	if cached, ok := tracePkgPathCache.Load(fileDir); ok {
		if s, ok := cached.(string); ok {
			return s
		}
	}

	searchDir := fileDir
	for {
		gomod := filepath.Join(searchDir, "go.mod")
		if data, err := os.ReadFile(gomod); err == nil {
			modulePath := parseModulePath(string(data))
			if modulePath == "" {
				break
			}

			rel, err := filepath.Rel(searchDir, fileDir)
			if err != nil {
				break
			}

			rel = filepath.ToSlash(rel)
			pkgPath := modulePath
			if rel != "." {
				pkgPath = modulePath + "/" + rel
			}

			tracePkgPathCache.Store(fileDir, pkgPath)
			return pkgPath
		}

		parent := filepath.Dir(searchDir)
		if parent == searchDir {
			break
		}
		searchDir = parent
	}

	tracePkgPathCache.Store(fileDir, "")
	return ""
}

func parseModulePath(goModContent string) string {
	for _, line := range strings.Split(goModContent, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
