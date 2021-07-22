package dix

import (
	"fmt"
	"reflect"

	"github.com/pubgo/xerror"
)

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

func (x *dix) dixPtr(values map[group][]reflect.Type, val reflect.Value) error {
	tye := getIndirectType(val.Type())
	if abcTy := x.getAbcType(tye); abcTy != nil {
		x.setAbcValue(abcTy, _default, tye)
	}

	x.setValue(tye, _default, val)
	values[_default] = append(values[_default], val.Type())
	return nil
}

func (x *dix) dixPtrInvoke(val reflect.Value) error {
	tye := getIndirectType(val.Type())
	x.getValue(tye, _default)
	val.Elem().Set(x.getValue(tye, _default))
	return nil
}

func (x *dix) dixStructInvoke(val reflect.Value) error {
	tye := val.Type()

	for i := 0; i < tye.NumField(); i++ {
		var kind = tye.Field(i).Type.Kind()
		if kind != reflect.Ptr && kind != reflect.Interface {
			continue
		}

		var ns = x.getNS(tye.Field(i))

		var retVal reflect.Value
		if kind == reflect.Ptr {
			retVal = x.getValue(getIndirectType(tye.Field(i).Type), ns)
		} else {
			retVal = x.getAbcValue(getIndirectType(tye.Field(i).Type), ns)
		}

		if !retVal.IsValid() || retVal.IsZero() {
			return fmt.Errorf("error")
		}

		val.Field(i).Elem().Set(retVal)
	}

	return nil
}
