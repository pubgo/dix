package dixinternal

import (
	"fmt"
	"github.com/samber/lo"
	"reflect"
	"time"

	"github.com/pubgo/funk/stack"
)

// FuncProvider 函数提供者实现
type FuncProvider struct {
	fn            reflect.Value
	primaryType   reflect.Type   // 主要返回类型
	providedTypes []reflect.Type // 能提供的所有类型
	dependencies  []Dependency
	initialized   bool
	isMap         bool
	isList        bool
	hasError      bool // 新增：标记是否返回 error
}

// NewFuncProvider 创建函数提供者
func NewFuncProvider(fn reflect.Value) (*FuncProvider, error) {
	if fn.Kind() != reflect.Func {
		return nil, NewValidationError("provider must be a function").
			WithDetail("actual_kind", fn.Kind().String())
	}

	fnType := fn.Type()
	if fnType.NumOut() == 0 {
		return nil, NewValidationError("provider function must have at least one return value")
	}

	if fnType.IsVariadic() {
		return nil, NewValidationError("provider function cannot have variadic parameters")
	}

	// 检查返回值：支持 (T) 或 (T, error) 两种形式
	hasError := false
	outputType := fnType.Out(0)

	switch {
	case fnType.NumOut() == 2:
		// 检查第二个返回值是否为 error 类型
		errorType := fnType.Out(1)
		if errorType.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			hasError = true
		} else {
			return nil, NewValidationError("second return value must be error type").
				WithDetail("actual_type", errorType.String())
		}
	case fnType.NumOut() > 2:
		return nil, NewValidationError("provider function can have at most 2 return values (value, error)").
			WithDetail("actual_count", fnType.NumOut())
	}

	// 解析依赖（包括输入参数和输出结构体字段）
	dependencies, err := parseDependencies(fnType)
	if err != nil {
		return nil, WrapError(err, ErrorTypeProvider, "failed to parse dependencies")
	}

	isMap := false
	isList := false

	// outputType 支持的类型是: []T, map[string]T, map[string][]T
	switch outputType.Kind() {
	case reflect.Map:
		isMap = true
		outputType = outputType.Elem()
		if outputType.Kind() == reflect.Slice {
			isList = true
			outputType = outputType.Elem()
		}

		if !isMapListSupportedType(outputType.Kind()) {
			return nil, NewValidationError("unsupported provider output type").
				WithDetail("type", outputType.String()).
				WithDetail("kind", outputType.Kind().String())
		}
	case reflect.Slice:
		isList = true
		outputType = outputType.Elem()
		if !isMapListSupportedType(outputType.Kind()) {
			return nil, NewValidationError("unsupported provider output type").
				WithDetail("type", outputType.String()).
				WithDetail("kind", outputType.Kind().String())
		}
	case reflect.Ptr, reflect.Interface, reflect.Func: // 支持的单类型
	case reflect.Struct: // 结构体类型，需要特殊处理
	default:
		return nil, NewValidationError("unsupported provider output type").
			WithDetail("type", outputType.String()).
			WithDetail("kind", outputType.Kind().String())
	}

	// 收集所有能提供的类型
	var providedTypes []reflect.Type

	// 如果输出类型是结构体，收集其字段类型作为可提供的类型
	// 注意：输出类型不是依赖，依赖只来自函数的输入参数
	if outputType.Kind() == reflect.Struct {
		// 收集结构体字段类型作为可提供的类型
		providedTypes = extractStructFieldTypes(outputType)
	} else {
		providedTypes = []reflect.Type{outputType}
	}

	return &FuncProvider{
		fn:            fn,
		primaryType:   outputType,
		providedTypes: providedTypes,
		dependencies:  dependencies,
		initialized:   false,
		isMap:         isMap,
		isList:        isList,
		hasError:      hasError,
	}, nil
}

// 实现新的 Provider 接口
func (p *FuncProvider) ProvidedTypes() []reflect.Type {
	return p.providedTypes
}

func (p *FuncProvider) PrimaryType() reflect.Type {
	return p.primaryType
}

func (p *FuncProvider) CanProvide(typ reflect.Type) bool {
	for _, provided := range p.providedTypes {
		if provided == typ {
			return true
		}
	}
	return false
}

// 保持向后兼容性
func (p *FuncProvider) Type() reflect.Type {
	return p.primaryType
}

// ProvideFor 为指定类型提供实例
func (p *FuncProvider) ProvideFor(typ reflect.Type, args []reflect.Value) (reflect.Value, error) {
	// 检查是否能提供该类型
	if !p.CanProvide(typ) {
		return reflect.Value{}, NewNotFoundError(typ).
			WithDetail("provider_primary_type", p.primaryType.String()).
			WithDetail("provider_provided_types", fmt.Sprintf("%v", p.providedTypes))
	}

	// 调用 provider 获取主要结果
	results, err := p.Invoke(args)
	if err != nil {
		return reflect.Value{}, WrapError(err, ErrorTypeProvider, "failed to invoke provider").
			WithDetail("target_type", typ.String()).
			WithDetail("provider_type", p.primaryType.String())
	}

	if len(results) == 0 {
		return reflect.Value{}, NewNotFoundError(typ).
			WithDetail("reason", "provider returned no results").
			WithDetail("provider_type", p.primaryType.String())
	}

	mainResult := results[0]

	// 如果请求的就是主要类型，直接返回
	if typ == p.primaryType {
		return mainResult, nil
	}

	// 如果主要结果是结构体，尝试从中提取字段
	if mainResult.Type().Kind() == reflect.Struct {
		fieldValue, err := p.extractFieldFromStruct(mainResult, typ)
		if err != nil {
			return reflect.Value{}, WrapError(err, ErrorTypeProvider, "failed to extract field from struct").
				WithDetail("target_type", typ.String()).
				WithDetail("struct_type", mainResult.Type().String()).
				WithDetail("provider_type", p.primaryType.String())
		}
		return fieldValue, nil
	}

	return reflect.Value{}, NewNotFoundError(typ).
		WithDetail("reason", "unable to provide type from provider result").
		WithDetail("provider_type", p.primaryType.String()).
		WithDetail("result_type", mainResult.Type().String())
}

// extractFieldFromStruct 从结构体中提取指定类型的字段
func (p *FuncProvider) extractFieldFromStruct(structValue reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	structType := structValue.Type()

	// 遍历结构体字段
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)

		// 跳过不导出的字段
		if !field.IsExported() {
			continue
		}

		// 精确类型匹配
		if field.Type == targetType {
			return fieldValue, nil
		}

		// 递归检查嵌套结构体
		if field.Type.Kind() == reflect.Struct {
			nestedValue, err := p.extractFieldFromStruct(fieldValue, targetType)
			if err == nil {
				return nestedValue, nil
			}
			// 继续查找其他字段，不返回错误
		}
	}

	return reflect.Value{}, NewNotFoundError(targetType).
		WithDetail("reason", "field type not found in struct").
		WithDetail("struct_type", structType.String())
}

func (p *FuncProvider) Invoke(args []reflect.Value) (results []reflect.Value, err error) {
	// 获取函数实际参数数量
	expectedArgCount := p.fn.Type().NumIn()
	if len(args) < expectedArgCount {
		return nil, NewInvocationError("insufficient arguments provided").
			WithDetail("expected", expectedArgCount).
			WithDetail("actual", len(args))
	}

	// 提取前N个参数用于函数调用（N=函数参数数量）
	fnArgs := args[:expectedArgCount]
	extraArgs := args[expectedArgCount:] // 剩余的参数用于结构体字段填充

	// 填充结构体参数的字段
	err = p.populateStructFields(fnArgs, extraArgs)
	if err != nil {
		return nil, WrapError(err, ErrorTypeInvocation, "failed to populate struct fields").
			WithDetail("provider_function", p.fn.Type().String())
	}

	// 收集入参信息用于错误记录
	argTypes := make([]string, len(fnArgs))
	argValues := make([]interface{}, len(fnArgs))
	for i, arg := range fnArgs {
		argTypes[i] = arg.Type().String()
		if arg.IsValid() && arg.CanInterface() {
			argValues[i] = arg.Interface()
		} else {
			argValues[i] = "<invalid_or_unexportable>"
		}
	}

	defer func() {
		if r := recover(); r != nil {
			// 记录调用栈信息
			fnStack := stack.CallerWithFunc(p.fn)
			logger.Error().
				Str("provider", fnStack.String()).
				Interface("panic", r).
				Strs("input_types", argTypes).
				Interface("input_values", argValues).
				Str("expected_output_type", p.primaryType.String()).
				Msg("provider function panicked")

			// 将 panic 转换为错误
			var panicErr error
			if e, ok := r.(error); ok {
				panicErr = e
			} else {
				panicErr = fmt.Errorf("panic: %v", r)
			}
			err = WrapError(panicErr, ErrorTypeInvocation, "provider function panicked").
				WithDetail("provider_type", p.primaryType.String()).
				WithDetail("panic_value", r).
				WithDetail("provider_location", fnStack.String()).
				WithDetail("input_types", argTypes).
				WithDetail("input_values", argValues).
				WithDetail("expected_output_type", p.primaryType.String())
		}
	}()

	start := time.Now()
	results = p.fn.Call(fnArgs)

	// 记录调用信息
	fnStack := stack.CallerWithFunc(p.fn)
	logger.Debug().
		Str("cost", time.Since(start).String()).
		Str("provider", fnStack.String()).
		Msgf("invoked provider %s", fnStack.Name)

	// 检查 error 返回值
	if p.hasError && len(results) >= 2 {
		errorValue := results[1]
		if !errorValue.IsNil() {
			// 提取 error 并返回
			if providerErr, ok := errorValue.Interface().(error); ok {
				// 记录 provider 返回 error 的详细信息
				fnStack := stack.CallerWithFunc(p.fn)
				logger.Error().
					Str("provider", fnStack.String()).
					Err(providerErr).
					Strs("input_types", argTypes).
					Interface("input_values", argValues).
					Str("expected_output_type", p.primaryType.String()).
					Msg("provider function returned error")

				return nil, WrapError(providerErr, ErrorTypeProvider, "provider function returned error").
					WithDetail("provider_type", p.primaryType.String()).
					WithDetail("provider_location", fnStack.String()).
					WithDetail("input_types", argTypes).
					WithDetail("input_values", argValues).
					WithDetail("expected_output_type", p.primaryType.String())
			}
		}
		// 如果有 error 返回值但为 nil，只返回第一个值
		return results[:1], nil
	}

	return results, nil
}

func (p *FuncProvider) Dependencies() []Dependency {
	return p.dependencies
}

func (p *FuncProvider) IsInitialized() bool {
	return p.initialized
}

func (p *FuncProvider) SetInitialized(initialized bool) {
	p.initialized = initialized
}

func (p *FuncProvider) IsMap() bool {
	return p.isMap
}

func (p *FuncProvider) IsList() bool {
	return p.isList
}

// DependencyImpl 依赖实现
type DependencyImpl struct {
	typ    reflect.Type
	isMap  bool
	isList bool
}

func NewDependency(typ reflect.Type, isMap, isList bool) *DependencyImpl {
	return &DependencyImpl{
		typ:    typ,
		isMap:  isMap,
		isList: isList,
	}
}

func (d *DependencyImpl) Type() reflect.Type {
	return d.typ
}

func (d *DependencyImpl) IsMap() bool {
	return d.isMap
}

func (d *DependencyImpl) IsList() bool {
	return d.isList
}

func (d *DependencyImpl) Validate() error {
	if d.isMap && !isSupportedType(d.typ.Kind()) {
		return NewValidationError("unsupported map value type").
			WithDetail("type", d.typ.String()).
			WithDetail("kind", d.typ.Kind().String())
	}

	if d.isList && !isSupportedType(d.typ.Kind()) {
		return NewValidationError("unsupported list element type").
			WithDetail("type", d.typ.String()).
			WithDetail("kind", d.typ.Kind().String())
	}

	if !isSupportedType(d.typ.Kind()) {
		return NewValidationError("unsupported dependency type").
			WithDetail("type", d.typ.String()).
			WithDetail("kind", d.typ.Kind().String())
	}

	return nil
}

// parseBasicDependencies 解析函数的基本依赖（不展开结构体字段）
func parseBasicDependencies(fnType reflect.Type) ([]Dependency, error) {
	var dependencies []Dependency

	for i := 0; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)

		switch paramType.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct:
			dep := NewDependency(paramType, false, false)
			if err := dep.Validate(); err != nil {
				return nil, err
			}
			dependencies = append(dependencies, dep)

		case reflect.Map:
			elemType := paramType.Elem()
			isList := elemType.Kind() == reflect.Slice
			if isList {
				elemType = elemType.Elem()
			}
			dep := NewDependency(elemType, true, isList)
			if err := dep.Validate(); err != nil {
				return nil, err
			}
			dependencies = append(dependencies, dep)

		case reflect.Slice:
			elemType := paramType.Elem()
			dep := NewDependency(elemType, false, true)
			if err := dep.Validate(); err != nil {
				return nil, err
			}
			dependencies = append(dependencies, dep)

		default:
			return nil, NewValidationError("unsupported parameter type").
				WithDetail("type", paramType.String()).
				WithDetail("kind", paramType.Kind().String()).
				WithDetail("parameter_index", i)
		}
	}

	return dependencies, nil
}

// parseDependencies 解析函数的依赖（包括结构体字段展开，用于provider）
func parseDependencies(fnType reflect.Type) ([]Dependency, error) {
	var dependencies []Dependency

	for i := 0; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)

		switch paramType.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func:
			dep := NewDependency(paramType, false, false)
			if err := dep.Validate(); err != nil {
				return nil, err
			}
			dependencies = append(dependencies, dep)

		case reflect.Struct:
			// 对于结构体，既添加结构体本身作为依赖，也添加其字段作为依赖
			dep := NewDependency(paramType, false, false)
			if err := dep.Validate(); err != nil {
				return nil, err
			}
			dependencies = append(dependencies, dep)

			// 递归解析结构体字段作为额外的依赖
			fieldDeps, err := parseStructFieldDependencies(paramType)
			if err != nil {
				return nil, WrapError(err, ErrorTypeProvider, "failed to parse struct field dependencies").
					WithDetail("struct_type", paramType.String()).
					WithDetail("parameter_index", i)
			}
			dependencies = append(dependencies, fieldDeps...)

		case reflect.Map:
			elemType := paramType.Elem()
			isList := elemType.Kind() == reflect.Slice
			if isList {
				elemType = elemType.Elem()
			}
			dep := NewDependency(elemType, true, isList)
			if err := dep.Validate(); err != nil {
				return nil, err
			}
			dependencies = append(dependencies, dep)

		case reflect.Slice:
			elemType := paramType.Elem()
			dep := NewDependency(elemType, false, true)
			if err := dep.Validate(); err != nil {
				return nil, err
			}
			dependencies = append(dependencies, dep)

		default:
			return nil, NewValidationError("unsupported parameter type").
				WithDetail("type", paramType.String()).
				WithDetail("kind", paramType.Kind().String()).
				WithDetail("parameter_index", i)
		}
	}

	return dependencies, nil
}

// parseStructFieldDependencies 递归解析结构体字段作为依赖
func parseStructFieldDependencies(structType reflect.Type) ([]Dependency, error) {
	var dependencies []Dependency

	// 使用map来避免重复依赖
	seen := make(map[reflect.Type]bool)

	err := parseStructFieldDependenciesRecursive(structType, &dependencies, seen)
	if err != nil {
		return nil, err
	}

	return dependencies, nil
}

// parseStructFieldDependenciesRecursive 递归解析结构体字段
func parseStructFieldDependenciesRecursive(structType reflect.Type, dependencies *[]Dependency, seen map[reflect.Type]bool) error {
	// 避免循环引用
	if seen[structType] {
		return nil
	}
	seen[structType] = true

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldType := field.Type

		// 跳过不导出的字段
		if !field.IsExported() {
			continue
		}

		// 跳过不支持注入的基本类型字段
		if !isInjectableFieldType(fieldType) {
			continue
		}

		switch fieldType.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func:
			// 避免重复添加相同类型的依赖
			if !seen[fieldType] {
				dep := NewDependency(fieldType, false, false)
				if err := dep.Validate(); err == nil { // 只添加有效的依赖
					*dependencies = append(*dependencies, dep)
					seen[fieldType] = true
				}
			}

		case reflect.Struct:
			// 递归处理嵌套结构体
			err := parseStructFieldDependenciesRecursive(fieldType, dependencies, seen)
			if err != nil {
				return WrapError(err, ErrorTypeProvider, "failed to parse nested struct dependencies").
					WithDetail("struct_type", structType.String()).
					WithDetail("field_name", field.Name).
					WithDetail("field_type", fieldType.String())
			}

		case reflect.Map:
			elemType := fieldType.Elem()
			isList := elemType.Kind() == reflect.Slice
			if isList {
				elemType = elemType.Elem()
			}
			// 避免重复添加相同类型的依赖
			mapKey := fieldType // 使用完整的map类型作为key
			if !seen[mapKey] {
				dep := NewDependency(elemType, true, isList)
				if err := dep.Validate(); err == nil {
					*dependencies = append(*dependencies, dep)
					seen[mapKey] = true
				}
			}

		case reflect.Slice:
			elemType := fieldType.Elem()
			// 避免重复添加相同类型的依赖
			sliceKey := fieldType // 使用完整的slice类型作为key
			if !seen[sliceKey] {
				dep := NewDependency(elemType, false, true)
				if err := dep.Validate(); err == nil {
					*dependencies = append(*dependencies, dep)
					seen[sliceKey] = true
				}
			}
		}
	}

	return nil
}

// populateStructFields 填充结构体参数的字段
func (p *FuncProvider) populateStructFields(fnArgs []reflect.Value, extraArgs []reflect.Value) error {
	if len(extraArgs) == 0 {
		return nil // 没有额外参数，无需填充
	}

	// 为每个额外参数建立类型到值的映射
	valueMap := make(map[reflect.Type]reflect.Value)
	for _, arg := range extraArgs {
		if arg.IsValid() {
			valueMap[arg.Type()] = arg
		}
	}

	// 遍历函数参数，查找结构体类型并填充其字段
	fnType := p.fn.Type()
	for i := 0; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)
		if paramType.Kind() == reflect.Struct && i < len(fnArgs) {
			err := p.populateStructValue(fnArgs[i], valueMap)
			if err != nil {
				return WrapError(err, ErrorTypeInvocation, "failed to populate struct parameter").
					WithDetail("parameter_index", i).
					WithDetail("parameter_type", paramType.String())
			}
		}
	}

	return nil
}

// populateStructValue 递归填充结构体值的字段
func (p *FuncProvider) populateStructValue(structValue reflect.Value, valueMap map[reflect.Type]reflect.Value) error {
	if structValue.Kind() != reflect.Struct {
		return nil
	}

	structType := structValue.Type()
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)

		// 跳过不可设置的字段
		if !fieldValue.CanSet() {
			continue
		}

		switch field.Type.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Func:
			// 查找匹配的值
			if val, exists := valueMap[field.Type]; exists && val.IsValid() {
				fieldValue.Set(val)
			}

		case reflect.Struct:
			// 递归处理嵌套结构体
			err := p.populateStructValue(fieldValue, valueMap)
			if err != nil {
				return WrapError(err, ErrorTypeInvocation, "failed to populate nested struct").
					WithDetail("struct_type", structType.String()).
					WithDetail("field_name", field.Name).
					WithDetail("field_type", field.Type.String())
			}

		case reflect.Map:
			// 处理Map字段
			elemType := field.Type.Elem()
			isList := elemType.Kind() == reflect.Slice
			if isList {
				elemType = elemType.Elem()
			}

			// 查找匹配类型的所有值来构建Map
			if p.hasMapValues(valueMap, elemType) {
				mapVal := p.buildMapFromValues(field.Type, valueMap, elemType, isList)
				if mapVal.IsValid() {
					fieldValue.Set(mapVal)
				}
			}

		case reflect.Slice:
			// 处理Slice字段
			elemType := field.Type.Elem()

			// 查找匹配类型的所有值来构建Slice
			if p.hasSliceValues(valueMap, elemType) {
				sliceVal := p.buildSliceFromValues(field.Type, valueMap, elemType)
				if sliceVal.IsValid() {
					fieldValue.Set(sliceVal)
				}
			}
		}
	}

	return nil
}

// hasMapValues 检查是否有构建Map所需的值
func (p *FuncProvider) hasMapValues(valueMap map[reflect.Type]reflect.Value, elemType reflect.Type) bool {
	for valType := range valueMap {
		if valType == elemType {
			return true
		}
	}
	return false
}

// hasSliceValues 检查是否有构建Slice所需的值
func (p *FuncProvider) hasSliceValues(valueMap map[reflect.Type]reflect.Value, elemType reflect.Type) bool {
	for valType := range valueMap {
		if valType == elemType {
			return true
		}
	}
	return false
}

// buildMapFromValues 从值映射构建Map
func (p *FuncProvider) buildMapFromValues(mapType reflect.Type, valueMap map[reflect.Type]reflect.Value, elemType reflect.Type, isList bool) reflect.Value {
	mapVal := reflect.MakeMap(mapType)

	for valType, val := range valueMap {
		if valType == elemType && val.IsValid() {
			// 使用类型名作为key
			key := reflect.ValueOf(valType.String())
			if isList {
				// 如果是列表，需要包装成slice
				sliceType := reflect.SliceOf(elemType)
				sliceVal := reflect.MakeSlice(sliceType, 0, 1)
				sliceVal = reflect.Append(sliceVal, val)
				mapVal.SetMapIndex(key, sliceVal)
			} else {
				mapVal.SetMapIndex(key, val)
			}
		}
	}

	return mapVal
}

// buildSliceFromValues 从值映射构建Slice
func (p *FuncProvider) buildSliceFromValues(sliceType reflect.Type, valueMap map[reflect.Type]reflect.Value, elemType reflect.Type) reflect.Value {
	sliceVal := reflect.MakeSlice(sliceType, 0, len(valueMap))

	for valType, val := range valueMap {
		if valType == elemType && val.IsValid() {
			sliceVal = reflect.Append(sliceVal, val)
		}
	}

	return sliceVal
}

// isSupportedType 检查是否为支持的类型
func isSupportedType(kind reflect.Kind) bool {
	switch kind {
	case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct, reflect.Map, reflect.Slice:
		return true
	default:
		return false
	}
}

func isMapListSupportedType(kind reflect.Kind) bool {
	switch kind {
	case reflect.Interface, reflect.Ptr, reflect.Func:
		return true
	default:
		return false
	}
}

// isInjectableFieldType 检查字段类型是否可注入
func isInjectableFieldType(fieldType reflect.Type) bool {
	switch fieldType.Kind() {
	case reflect.Interface, reflect.Ptr, reflect.Func, reflect.Struct, reflect.Map, reflect.Slice:
		return true
	default:
		return false
	}
}

// extractStructFieldTypes 提取结构体的所有导出字段类型
func extractStructFieldTypes(structType reflect.Type) []reflect.Type {
	var types []reflect.Type

	if structType.Kind() != reflect.Struct {
		return types
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() || !isInjectableFieldType(field.Type) {
			continue
		}

		types = append(types, field.Type)

		// 递归处理嵌套结构体
		if field.Type.Kind() == reflect.Struct {
			nestedTypes := extractStructFieldTypes(field.Type)
			for _, nestedType := range nestedTypes {
				types = append(types, nestedType)
			}
		}
	}

	return lo.Uniq(types)
}
