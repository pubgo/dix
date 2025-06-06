package dixinternal

import (
	"reflect"
	"strings"
)

// InjectorImpl 注入器实现
type InjectorImpl struct {
	resolver Resolver
}

// NewInjector 创建新的注入器
func NewInjector(resolver Resolver) *InjectorImpl {
	return &InjectorImpl{
		resolver: resolver,
	}
}

// InjectStruct 注入结构体
func (inj *InjectorImpl) InjectStruct(target reflect.Value, opts Options) error {
	if target.Kind() != reflect.Struct {
		return NewValidationError("target must be a struct").
			WithDetail("actual_kind", target.Kind().String())
	}

	typ := target.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := target.Field(i)

		// 跳过不可设置的字段
		if !fieldValue.CanSet() && field.Type.Kind() != reflect.Struct {
			logger.Debug().Msgf("skipping unsettable field: %s", field.Name)
			continue
		}

		err := inj.injectField(fieldValue, field, opts)
		if err != nil {
			return WrapError(err, ErrorTypeInjection, "failed to inject struct field").
				WithDetail("struct_type", typ.String()).
				WithDetail("field_name", field.Name).
				WithDetail("field_type", field.Type.String())
		}
	}

	return nil
}

// InjectFunc 注入函数
func (inj *InjectorImpl) InjectFunc(fn reflect.Value, opts Options) error {
	if fn.Kind() != reflect.Func {
		return NewValidationError("target must be a function").
			WithDetail("actual_kind", fn.Kind().String())
	}

	fnType := fn.Type()
	if fnType.NumOut() != 0 {
		return NewValidationError("injectable function must have no return values").
			WithDetail("return_count", fnType.NumOut())
	}

	if fnType.NumIn() == 0 {
		return NewValidationError("injectable function must have at least one parameter").
			WithDetail("parameter_count", fnType.NumIn())
	}

	// 解析函数参数依赖
	dependencies, err := parseDependencies(fnType)
	if err != nil {
		return WrapError(err, ErrorTypeValidation, "failed to parse function dependencies")
	}

	// 解析所有依赖
	args, err := inj.resolver.ResolveAll(dependencies, opts)
	if err != nil {
		return WrapError(err, ErrorTypeInjection, "failed to resolve function dependencies")
	}

	// 调用函数
	fn.Call(args)
	return nil
}

// InjectTarget 注入目标对象（统一入口）
func (inj *InjectorImpl) InjectTarget(target interface{}, opts Options) error {
	if target == nil {
		return NewValidationError("injection target cannot be nil")
	}

	targetValue := reflect.ValueOf(target)
	if !targetValue.IsValid() || targetValue.IsNil() {
		return NewValidationError("injection target must be valid and non-nil")
	}

	// 处理函数类型
	if targetValue.Kind() == reflect.Func {
		return inj.InjectFunc(targetValue, opts)
	}

	// 处理指针类型
	if targetValue.Kind() != reflect.Ptr {
		return NewValidationError("injection target must be a pointer").
			WithDetail("actual_kind", targetValue.Kind().String())
	}

	// 注入方法（以DixInject开头的方法）
	err := inj.injectMethods(targetValue, opts)
	if err != nil {
		return err
	}

	// 解引用到实际值
	for targetValue.Kind() == reflect.Ptr {
		targetValue = targetValue.Elem()
	}

	// 处理结构体
	if targetValue.Kind() != reflect.Struct {
		return NewValidationError("injection target must point to a struct").
			WithDetail("actual_kind", targetValue.Kind().String())
	}

	return inj.InjectStruct(targetValue, opts)
}

// injectField 注入结构体字段
func (inj *InjectorImpl) injectField(fieldValue reflect.Value, field reflect.StructField, opts Options) error {
	switch field.Type.Kind() {
	case reflect.Struct:
		// 递归注入嵌套结构体
		return inj.InjectStruct(fieldValue, opts)

	case reflect.Interface, reflect.Ptr, reflect.Func:
		// 解析并设置值
		val, err := inj.resolver.Resolve(field.Type, opts)
		if err != nil {
			return err
		}
		if val.IsValid() {
			fieldValue.Set(val)
		}

	case reflect.Map:
		// 处理Map类型
		elemType := field.Type.Elem()
		isList := elemType.Kind() == reflect.Slice
		if isList {
			elemType = elemType.Elem()
		}

		dep := NewDependency(elemType, true, isList)
		val, err := inj.resolveAsMap(dep, opts)
		if err != nil {
			return err
		}
		if val.IsValid() {
			fieldValue.Set(val)
		}

	case reflect.Slice:
		// 处理Slice类型
		elemType := field.Type.Elem()
		dep := NewDependency(elemType, false, true)
		val, err := inj.resolveAsList(dep, opts)
		if err != nil {
			return err
		}
		if val.IsValid() {
			fieldValue.Set(val)
		}

	default:
		return NewValidationError("unsupported field type for injection").
			WithDetail("field_name", field.Name).
			WithDetail("field_type", field.Type.String()).
			WithDetail("field_kind", field.Type.Kind().String())
	}

	return nil
}

// injectMethods 注入方法
func (inj *InjectorImpl) injectMethods(target reflect.Value, opts Options) error {
	targetType := target.Type()

	for i := 0; i < target.NumMethod(); i++ {
		method := targetType.Method(i)

		// 只处理以InjectMethodPrefix开头的方法
		if !strings.HasPrefix(method.Name, InjectMethodPrefix) {
			continue
		}

		methodValue := target.Method(i)
		err := inj.InjectFunc(methodValue, opts)
		if err != nil {
			return WrapError(err, ErrorTypeInjection, "failed to inject method").
				WithDetail("method_name", method.Name).
				WithDetail("target_type", targetType.String())
		}
	}

	return nil
}

// resolveAsMap 解析为Map（辅助方法）
func (inj *InjectorImpl) resolveAsMap(dep Dependency, opts Options) (reflect.Value, error) {
	if resolver, ok := inj.resolver.(*ResolverImpl); ok {
		return resolver.resolveAsMap(dep, opts)
	}

	// 如果不是ResolverImpl，使用通用方法
	val, err := inj.resolver.Resolve(dep.Type(), opts)
	if err != nil {
		return reflect.Value{}, err
	}

	// 创建包含单个值的Map
	mapType := reflect.MapOf(reflect.TypeOf(""), dep.Type())
	mapVal := reflect.MakeMap(mapType)
	mapVal.SetMapIndex(reflect.ValueOf(defaultKey), val)

	return mapVal, nil
}

// resolveAsList 解析为List（辅助方法）
func (inj *InjectorImpl) resolveAsList(dep Dependency, opts Options) (reflect.Value, error) {
	if resolver, ok := inj.resolver.(*ResolverImpl); ok {
		return resolver.resolveAsList(dep, opts)
	}

	// 如果不是ResolverImpl，使用通用方法
	val, err := inj.resolver.Resolve(dep.Type(), opts)
	if err != nil {
		return reflect.Value{}, err
	}

	// 创建包含单个值的Slice
	sliceType := reflect.SliceOf(dep.Type())
	sliceVal := reflect.MakeSlice(sliceType, 0, 1)
	sliceVal = reflect.Append(sliceVal, val)

	return sliceVal, nil
}
