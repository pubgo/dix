package dixinternal

import (
	"reflect"
	"strings"

	"github.com/pubgo/funk/log"
)

const (
	// defaultKey default namespace
	defaultKey = "default"

	// InjectMethodPrefix can inject objects, as long as the method of this object contains a prefix of `InjectMethodPrefix`
	InjectMethodPrefix = "DixInject"
)

var logger = log.GetLogger("dix")

func SetLog(setter func(logger log.Logger) log.Logger) {
	logger = setter(logger)
}

// ResolverImpl 依赖解析器实现
type ResolverImpl struct {
	providers map[reflect.Type][]Provider
	objects   map[reflect.Type]map[string][]reflect.Value
}

// NewResolver 创建新的解析器
func NewResolver() *ResolverImpl {
	return &ResolverImpl{
		providers: make(map[reflect.Type][]Provider),
		objects:   make(map[reflect.Type]map[string][]reflect.Value),
	}
}

// AddProvider 添加提供者
func (r *ResolverImpl) AddProvider(provider Provider) {
	typ := provider.Type()
	r.providers[typ] = append(r.providers[typ], provider)
}

// GetProviders 获取指定类型的提供者
func (r *ResolverImpl) GetProviders(typ reflect.Type) []Provider {
	return r.providers[typ]
}

// Resolve 解析单个依赖
func (r *ResolverImpl) Resolve(typ reflect.Type, opts Options) (reflect.Value, error) {
	// 处理结构体类型
	if typ.Kind() == reflect.Struct {
		return r.resolveStruct(typ, opts)
	}

	// 获取类型的所有值
	values, err := r.getTypeValues(typ, opts)
	if err != nil {
		return reflect.Value{}, err
	}

	// 返回最后一个值作为默认值
	if defaultValues, ok := values[defaultKey]; ok && len(defaultValues) > 0 {
		val := defaultValues[len(defaultValues)-1]
		if val.IsZero() && !opts.AllowValuesNull {
			return reflect.Value{}, NewNotFoundError(typ).
				WithDetail("reason", "resolved value is nil")
		}
		return val, nil
	}

	if !opts.AllowValuesNull {
		return reflect.Value{}, NewNotFoundError(typ)
	}

	return reflect.Zero(typ), nil
}

// ResolveAll 解析所有依赖
func (r *ResolverImpl) ResolveAll(deps []Dependency, opts Options) ([]reflect.Value, error) {
	var results []reflect.Value

	for _, dep := range deps {
		var val reflect.Value
		var err error

		switch {
		case dep.IsMap():
			val, err = r.resolveAsMap(dep, opts)
		case dep.IsList():
			val, err = r.resolveAsList(dep, opts)
		default:
			val, err = r.Resolve(dep.Type(), opts)
		}

		if err != nil {
			return nil, err
		}

		results = append(results, val)
	}

	return results, nil
}

// resolveStruct 解析结构体
func (r *ResolverImpl) resolveStruct(typ reflect.Type, opts Options) (reflect.Value, error) {
	val := reflect.New(typ)
	structVal := val.Elem()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !structVal.Field(i).CanSet() {
			continue
		}

		fieldVal, err := r.resolveField(field, opts)
		if err != nil {
			return reflect.Value{}, WrapError(err, ErrorTypeInjection, "failed to resolve struct field").
				WithDetail("struct_type", typ.String()).
				WithDetail("field_name", field.Name).
				WithDetail("field_type", field.Type.String())
		}

		if fieldVal.IsValid() {
			structVal.Field(i).Set(fieldVal)
		}
	}

	return structVal, nil
}

// resolveField 解析结构体字段
func (r *ResolverImpl) resolveField(field reflect.StructField, opts Options) (reflect.Value, error) {
	switch field.Type.Kind() {
	case reflect.Struct:
		return r.resolveStruct(field.Type, opts)
	case reflect.Interface, reflect.Ptr, reflect.Func:
		return r.Resolve(field.Type, opts)
	case reflect.Map:
		dep := NewDependency(field.Type.Elem(), true, field.Type.Elem().Kind() == reflect.Slice)
		if field.Type.Elem().Kind() == reflect.Slice {
			dep = NewDependency(field.Type.Elem().Elem(), true, true)
		}
		return r.resolveAsMap(dep, opts)
	case reflect.Slice:
		dep := NewDependency(field.Type.Elem(), false, true)
		return r.resolveAsList(dep, opts)
	default:
		return reflect.Value{}, NewValidationError("unsupported field type").
			WithDetail("field_name", field.Name).
			WithDetail("field_type", field.Type.String()).
			WithDetail("field_kind", field.Type.Kind().String())
	}
}

// resolveAsMap 解析为Map类型
func (r *ResolverImpl) resolveAsMap(dep Dependency, opts Options) (reflect.Value, error) {
	values, err := r.getTypeValues(dep.Type(), opts)
	if err != nil {
		return reflect.Value{}, err
	}

	if !opts.AllowValuesNull && len(values) == 0 {
		return reflect.Value{}, NewNotFoundError(dep.Type()).
			WithDetail("resolve_type", "map")
	}

	return r.makeMap(dep.Type(), values, dep.IsList()), nil
}

// resolveAsList 解析为List类型
func (r *ResolverImpl) resolveAsList(dep Dependency, opts Options) (reflect.Value, error) {
	values, err := r.getTypeValues(dep.Type(), opts)
	if err != nil {
		return reflect.Value{}, err
	}

	defaultValues := values[defaultKey]
	if !opts.AllowValuesNull && len(defaultValues) == 0 {
		return reflect.Value{}, NewNotFoundError(dep.Type()).
			WithDetail("resolve_type", "list")
	}

	return r.makeList(dep.Type(), defaultValues), nil
}

// getTypeValues 获取类型的所有值
func (r *ResolverImpl) getTypeValues(typ reflect.Type, opts Options) (map[string][]reflect.Value, error) {
	// 检查是否已有缓存的对象
	if r.objects[typ] == nil {
		r.objects[typ] = make(map[string][]reflect.Value)
	}

	// 如果没有提供者，发出警告
	if len(r.providers[typ]) == 0 {
		logger.Warn().
			Str("type", typ.String()).
			Str("kind", typ.Kind().String()).
			Msg("no providers found for type")
		return r.objects[typ], nil
	}

	// 调用所有未初始化的提供者
	for _, provider := range r.providers[typ] {
		if provider.IsInitialized() {
			continue
		}

		// 解析提供者的依赖
		args, err := r.ResolveAll(provider.Dependencies(), opts)
		if err != nil {
			return nil, WrapError(err, ErrorTypeProvider, "failed to resolve provider dependencies").
				WithDetail("provider_type", typ.String())
		}

		// 调用提供者
		results, err := provider.Invoke(args)
		if err != nil {
			return nil, WrapError(err, ErrorTypeInvocation, "provider invocation failed").
				WithDetail("provider_type", typ.String())
		}

		// 处理提供者的输出
		if len(results) > 0 {
			err = r.handleProviderOutput(typ, results[0], provider)
			if err != nil {
				return nil, err
			}
		}

		provider.SetInitialized(true)
	}

	return r.objects[typ], nil
}

// handleProviderOutput 处理提供者输出
func (r *ResolverImpl) handleProviderOutput(outputType reflect.Type, result reflect.Value, provider Provider) error {
	if !result.IsValid() || result.IsZero() {
		return nil
	}

	objects := r.extractValues(outputType, result, provider)

	// 合并到现有对象中
	for typ, groupValues := range objects {
		if r.objects[typ] == nil {
			r.objects[typ] = make(map[string][]reflect.Value)
		}

		for group, values := range groupValues {
			r.objects[typ][group] = append(r.objects[typ][group], values...)
		}
	}

	return nil
}

// extractValues 从提供者结果中提取值
func (r *ResolverImpl) extractValues(outputType reflect.Type, result reflect.Value, provider Provider) map[reflect.Type]map[string][]reflect.Value {
	objects := make(map[reflect.Type]map[string][]reflect.Value)

	switch result.Kind() {
	case reflect.Map:
		r.extractMapValues(outputType, result, provider, objects)
	case reflect.Slice:
		r.extractSliceValues(outputType, result, objects)
	case reflect.Struct:
		r.extractStructValues(result, objects)
	default:
		r.extractSingleValue(outputType, result, objects)
	}

	return objects
}

// extractMapValues 提取Map值
func (r *ResolverImpl) extractMapValues(outputType reflect.Type, result reflect.Value, provider Provider, objects map[reflect.Type]map[string][]reflect.Value) {
	elemType := result.Type().Elem()
	isList := elemType.Kind() == reflect.Slice
	if isList {
		elemType = elemType.Elem()
	}

	if objects[elemType] == nil {
		objects[elemType] = make(map[string][]reflect.Value)
	}

	for _, key := range result.MapKeys() {
		keyStr := strings.TrimSpace(key.String())
		if keyStr == "" {
			keyStr = defaultKey
		}

		val := result.MapIndex(key)
		if !val.IsValid() || val.IsNil() {
			continue
		}

		if isList {
			for i := 0; i < val.Len(); i++ {
				item := val.Index(i)
				if item.IsValid() && !item.IsNil() {
					objects[elemType][keyStr] = append(objects[elemType][keyStr], item)
				}
			}
		} else {
			objects[elemType][keyStr] = append(objects[elemType][keyStr], val)
		}
	}
}

// extractSliceValues 提取Slice值
func (r *ResolverImpl) extractSliceValues(outputType reflect.Type, result reflect.Value, objects map[reflect.Type]map[string][]reflect.Value) {
	elemType := result.Type().Elem()

	if objects[elemType] == nil {
		objects[elemType] = make(map[string][]reflect.Value)
	}

	for i := 0; i < result.Len(); i++ {
		val := result.Index(i)
		if val.IsValid() && !val.IsNil() {
			objects[elemType][defaultKey] = append(objects[elemType][defaultKey], val)
		}
	}
}

// extractStructValues 提取结构体值
func (r *ResolverImpl) extractStructValues(result reflect.Value, objects map[reflect.Type]map[string][]reflect.Value) {
	for i := 0; i < result.NumField(); i++ {
		field := result.Field(i)
		fieldObjects := r.extractValues(field.Type(), field, nil)

		for typ, groupValues := range fieldObjects {
			if objects[typ] == nil {
				objects[typ] = groupValues
			} else {
				for group, values := range groupValues {
					objects[typ][group] = append(objects[typ][group], values...)
				}
			}
		}
	}
}

// extractSingleValue 提取单个值
func (r *ResolverImpl) extractSingleValue(outputType reflect.Type, result reflect.Value, objects map[reflect.Type]map[string][]reflect.Value) {
	if objects[outputType] == nil {
		objects[outputType] = make(map[string][]reflect.Value)
	}

	if result.IsValid() && !result.IsNil() {
		objects[outputType][defaultKey] = []reflect.Value{result}
	}
}

// makeMap 创建Map
func (r *ResolverImpl) makeMap(typ reflect.Type, data map[string][]reflect.Value, valueList bool) reflect.Value {
	var mapType reflect.Type
	if valueList {
		mapType = reflect.MapOf(reflect.TypeOf(""), reflect.SliceOf(typ))
	} else {
		mapType = reflect.MapOf(reflect.TypeOf(""), typ)
	}

	mapVal := reflect.MakeMap(mapType)
	for key, values := range data {
		if len(values) == 0 {
			continue
		}

		var val reflect.Value
		if valueList {
			sliceVal := reflect.MakeSlice(reflect.SliceOf(typ), 0, len(values))
			val = reflect.Append(sliceVal, values...)
		} else {
			val = values[len(values)-1] // 使用最后一个值
		}

		mapVal.SetMapIndex(reflect.ValueOf(key), val)
	}

	return mapVal
}

// makeList 创建List
func (r *ResolverImpl) makeList(typ reflect.Type, data []reflect.Value) reflect.Value {
	sliceVal := reflect.MakeSlice(reflect.SliceOf(typ), 0, len(data))
	return reflect.Append(sliceVal, data...)
}
