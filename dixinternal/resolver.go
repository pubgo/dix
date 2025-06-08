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
	resolving map[reflect.Type]bool // 用于防止递归解析同一类型
}

// NewResolver 创建新的解析器
func NewResolver() *ResolverImpl {
	return &ResolverImpl{
		providers: make(map[reflect.Type][]Provider),
		objects:   make(map[reflect.Type]map[string][]reflect.Value),
		resolving: make(map[reflect.Type]bool),
	}
}

// AddProvider 添加提供者
func (r *ResolverImpl) AddProvider(provider Provider) {
	// 为 provider 能提供的所有类型注册
	providedTypes := provider.ProvidedTypes()

	for _, typ := range providedTypes {
		r.providers[typ] = append(r.providers[typ], provider)
	}

	// 向后兼容：如果没有 ProvidedTypes，使用传统的 PrimaryType() 方法
	if len(providedTypes) == 0 {
		typ := provider.PrimaryType()
		r.providers[typ] = append(r.providers[typ], provider)
	}
}

// GetProviders 获取指定类型的提供者
func (r *ResolverImpl) GetProviders(typ reflect.Type) []Provider {
	return r.providers[typ]
}

// Resolve 解析单个依赖
func (r *ResolverImpl) Resolve(typ reflect.Type, opts Options) (reflect.Value, error) {
	// 优先尝试通过 provider 获取类型的值
	values, err := r.getTypeValues(typ, opts)
	if err != nil {
		return reflect.Value{}, err
	}

	// 如果找到了 provider 提供的值，返回最后一个值作为默认值
	if defaultValues, ok := values[defaultKey]; ok && len(defaultValues) > 0 {
		val := defaultValues[len(defaultValues)-1]
		if val.IsZero() && !opts.AllowValuesNull {
			return reflect.Value{}, NewNotFoundError(typ).
				WithDetail("reason", "resolved value is nil")
		}
		return val, nil
	}

	// 如果没有找到 provider，且是结构体类型，才尝试通过字段注入创建
	if typ.Kind() == reflect.Struct {
		return r.resolveStruct(typ, opts)
	}

	// 其他情况：没有 provider 且不是结构体
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
	// 防止递归解析同一类型
	if r.resolving[typ] {
		return map[string][]reflect.Value{}, nil // 返回空结果，避免死循环
	}

	// 检查是否已有缓存的对象
	if r.objects[typ] == nil {
		r.objects[typ] = make(map[string][]reflect.Value)
	}

	// 如果已有缓存，直接返回
	if len(r.objects[typ][defaultKey]) > 0 {
		return r.objects[typ], nil
	}

	// 标记正在解析
	r.resolving[typ] = true
	defer func() {
		delete(r.resolving, typ)
	}()

	// 首先尝试调用直接的提供者
	if len(r.providers[typ]) > 0 {
		err := r.invokeDirectProviders(typ, opts)
		if err != nil {
			return nil, err
		}
	}

	// 如果还没有找到该类型的值，尝试从其他提供者的输出中查找
	if len(r.objects[typ][defaultKey]) == 0 {
		err := r.invokeIndirectProviders(typ, opts)
		if err != nil {
			return nil, err
		}
	}

	// 如果仍然没有找到，发出警告
	if len(r.objects[typ][defaultKey]) == 0 {
		logger.Warn().
			Str("type", typ.String()).
			Str("kind", typ.Kind().String()).
			Msg("no providers found for type")
	}

	return r.objects[typ], nil
}

// invokeDirectProviders 调用直接的提供者
func (r *ResolverImpl) invokeDirectProviders(typ reflect.Type, opts Options) error {
	for _, provider := range r.providers[typ] {
		// 检查 provider 是否能提供该类型
		if provider.CanProvide(typ) {
			// 对于多类型 provider，检查是否已经有该类型的缓存
			if r.objects[typ] != nil && len(r.objects[typ][defaultKey]) > 0 {
				continue // 已经有缓存，跳过
			}

			err := r.invokeProviderForType(provider, typ, opts)
			if err != nil {
				return err
			}
		} else {
			// 传统单类型 provider
			if provider.IsInitialized() {
				continue
			}

			err := r.invokeProvider(provider, typ, opts)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// invokeProviderForType 为特定类型调用提供者
func (r *ResolverImpl) invokeProviderForType(provider Provider, targetType reflect.Type, opts Options) error {
	wasInitialized := provider.IsInitialized()

	// 如果还没初始化，先标记为已初始化，避免无限递归
	if !wasInitialized {
		provider.SetInitialized(true)
	}

	// 解析提供者的依赖
	args, err := r.ResolveAll(provider.Dependencies(), opts)
	if err != nil {
		return WrapError(err, ErrorTypeProvider, "failed to resolve provider dependencies").
			WithDetail("provider_type", provider.PrimaryType().String()).
			WithDetail("target_type", targetType.String())
	}

	// 使用 ProvideFor 方法为特定类型提供实例
	result, err := provider.ProvideFor(targetType, args)
	if err != nil {
		return WrapError(err, ErrorTypeInvocation, "provider failed to provide for type").
			WithDetail("provider_type", provider.PrimaryType().String()).
			WithDetail("target_type", targetType.String())
	}

	// 直接将结果添加到对象缓存中
	if r.objects[targetType] == nil {
		r.objects[targetType] = make(map[string][]reflect.Value)
	}

	if result.IsValid() && !result.IsZero() {
		r.objects[targetType][defaultKey] = append(r.objects[targetType][defaultKey], result)
	}

	return nil
}

// invokeIndirectProviders 从其他提供者的输出中查找类型
func (r *ResolverImpl) invokeIndirectProviders(typ reflect.Type, opts Options) error {
	// 遍历所有其他类型的提供者
	for providerType, providers := range r.providers {
		if providerType == typ {
			continue // 跳过我们已经处理过的直接提供者
		}

		for _, provider := range providers {
			if provider.IsInitialized() {
				continue
			}

			err := r.invokeProvider(provider, providerType, opts)
			if err != nil {
				return err
			}

			// 不要在找到第一个匹配后就返回，继续调用所有提供者
			// 这样可以收集到所有可能产生该类型的实例
		}
	}
	return nil
}

// invokeProvider 调用单个提供者
func (r *ResolverImpl) invokeProvider(provider Provider, providerType reflect.Type, opts Options) error {
	// 解析提供者的依赖
	args, err := r.ResolveAll(provider.Dependencies(), opts)
	if err != nil {
		return WrapError(err, ErrorTypeProvider, "failed to resolve provider dependencies").
			WithDetail("provider_type", providerType.String())
	}

	// 调用提供者
	results, err := provider.Invoke(args)
	if err != nil {
		return WrapError(err, ErrorTypeInvocation, "provider invocation failed").
			WithDetail("provider_type", providerType.String())
	}

	// 处理提供者的输出
	if len(results) > 0 {
		err = r.handleProviderOutput(providerType, results[0], provider)
		if err != nil {
			return err
		}
	}

	provider.SetInitialized(true)
	return nil
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
		fieldType := result.Type().Field(i)

		// 跳过不导出的字段
		if !fieldType.IsExported() {
			continue
		}

		// 跳过无效字段
		if !field.IsValid() {
			continue
		}

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

	// 检查值是否有效
	if !result.IsValid() {
		return
	}

	// 对于可以为 nil 的类型，检查是否为 nil
	canBeNil := result.Kind() == reflect.Ptr ||
		result.Kind() == reflect.Interface ||
		result.Kind() == reflect.Slice ||
		result.Kind() == reflect.Map ||
		result.Kind() == reflect.Chan ||
		result.Kind() == reflect.Func

	if canBeNil && result.IsNil() {
		return
	}

	// 对于基本类型，检查是否为零值（可选）
	// 这里我们允许零值，因为零值也是有效的值
	objects[outputType][defaultKey] = []reflect.Value{result}
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
