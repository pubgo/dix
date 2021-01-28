package dix

import (
	"fmt"
	"reflect"

	"github.com/pubgo/xerror"
)

// dix(struct)
// 输入参数是struct类型, 结构体类型中的每一个属性都要是指针类型
func (x *dix) handleStruct(data interface{}) (values map[group][]key, gErr error) {
	defer xerror.Resp(func(err xerror.XErr) { gErr = err.WrapF("[dix] unknown error, data:%#v", data) })

	values = make(map[group][]key)

	sVal := reflect.ValueOf(data)
	sTyp := sVal.Type()

	// 结构体类型检查
	xerror.Assert(sTyp.Kind() != reflect.Struct, "[data] %s should be struct type", sTyp.Name())

	for i := 0; i < sTyp.NumField(); i++ {
		fVal := sVal.Field(i)
		fTyp := sTyp.Field(i)

		ftInfo := func() string { return fmt.Sprintf("the struct[%s] field[%s]", sTyp.Name(), fTyp.Name) }

		// 检查类型是否是指针类型
		if fVal.Kind() != reflect.Ptr {
			return nil, xerror.WrapF(Err, "%s should be Ptr type", ftInfo())
		}

		// 检查是否是指针的指针类型`**ptr`
		if isDoublePtr(fVal.Type()) {
			return nil, xerror.WrapF(Err, "%s should not be double Ptr type", ftInfo())
		}

		// 检查是否是空指针
		if x.isNil(fVal) {
			return nil, xerror.WrapF(Err, "%s should not be nil", ftInfo())
		}

		// 检查类型是否某个接口的实现
		ft := getIndirectType(fVal.Type())
		if ttk := x.getAbcType(ft); ttk != nil {
			x.setAbcValue(ttk, x.getNS(fTyp), ft)
		}

		x.setValue(ft, x.getNS(fTyp), fVal)
		values[x.getNS(fTyp)] = append(values[x.getNS(fTyp)], fVal.Type())
	}

	return values, nil
}

// dix(map)
// 输入参数是map类型, map类型中的每一个key都是group, value都是ptr value
func (x *dix) handleMap(data interface{}) (values map[group][]key, gErr error) {
	defer xerror.Resp(func(err xerror.XErr) { gErr = err.WrapF("[dix] unknown error, data:%#v", data) })

	values = make(map[group][]key)

	sVal := reflect.ValueOf(data)
	sTyp := sVal.Type()

	iter := sVal.MapRange()
	for iter.Next() {
		k := iter.Key().String()
		if k == "" {
			return nil, xerror.New("map key is null")
		}

		if iter.Value().Type().Kind() != reflect.Ptr {
			return nil, xerror.WrapF(Err, "key %v should be Ptr type", iter.Key().Interface())
		}

		if x.isNil(iter.Value()) {
			return nil, xerror.Fmt("map value is nil, key:%s", k)
		}

		if ttk := x.getAbcType(getIndirectType(iter.Value().Type())); ttk != nil {
			x.setAbcValue(ttk, k, getIndirectType(iter.Value().Type()))
		}

		x.setValue(getIndirectType(iter.Value().Type()), k, iter.Value())
		values[k] = append(values[k], iter.Value().Type())
	}

	// 结构体类型检查
	xerror.Assert(sTyp.Kind() != reflect.Map, "[data] %s should be map type", sTyp.Name())

	return values, nil
}

// dix(ptr)
func (x *dix) handlePtr(data interface{}) (values map[group][]key, gErr error) {
	defer xerror.Resp(func(err xerror.XErr) { gErr = err.WrapF("[dix] unknown error, data:%#v", data) })

	values = make(map[group][]key)

	sVal := reflect.ValueOf(data)
	sTyp := sVal.Type()

	// 结构体类型检查
	xerror.Assert(sTyp.Kind() != reflect.Ptr, "[data] %s should be ptr type", sTyp.Name())

	return values, nil
}

func (x *dix) dixMap(values map[group][]reflect.Type, data interface{}) error {
	val := reflect.ValueOf(data)

	if val.Type().Key().Kind() != reflect.String {
		return xerror.New("the map key should be string type")
	}

	iter := val.MapRange()
	for iter.Next() {
		if iter.Value().Type().Kind() != reflect.Ptr {
			return xerror.New("the map value should be Ptr type")
		}

		k := iter.Key().String()
		if k == "" {
			return xerror.New("map key is null")
		}

		if x.isNil(iter.Value()) {
			return xerror.Fmt("map value is nil, key:%s", k)
		}

		if ttk := x.getAbcType(getIndirectType(iter.Value().Type())); ttk != nil {
			x.setAbcValue(ttk, k, getIndirectType(iter.Value().Type()))
		}

		x.setValue(getIndirectType(iter.Value().Type()), k, iter.Value())
		values[k] = append(values[k], iter.Value().Type())
	}

	return nil
}

func (x *dix) dixStruct(values map[group][]reflect.Type, data interface{}) error {
	val := reflect.ValueOf(data)
	tye := val.Type()

	for i := 0; i < tye.NumField(); i++ {
		if tye.Field(i).Type.Kind() != reflect.Ptr {
			return xerror.New("the struct field should be Ptr type")
		}

		if x.isNil(val.Field(i)) {
			return xerror.New("struct field data is nil")
		}

		if ttk := x.getAbcType(getIndirectType(tye.Field(i).Type)); ttk != nil {
			x.setAbcValue(ttk, x.getNS(tye.Field(i)), getIndirectType(tye.Field(i).Type))
		}
		x.setValue(getIndirectType(tye.Field(i).Type), x.getNS(tye.Field(i)), val.Field(i))
		values[x.getNS(tye.Field(i))] = append(values[x.getNS(tye.Field(i))], val.Field(i).Type())
	}

	return nil
}

func (x *dix) dixPtr(values map[group][]reflect.Type, data interface{}) error {
	val := reflect.ValueOf(data)
	if x.isNil(val) {
		return xerror.New("data is nil")
	}

	tye := getIndirectType(val.Type())
	if ttk := x.getAbcType(tye); ttk != nil {
		x.setAbcValue(ttk, _default, tye)
	}

	x.setValue(tye, _default, val)
	values[_default] = append(values[_default], val.Type())
	return nil
}
