package dix

import (
	"reflect"

	"github.com/pubgo/xerror"
)

func (x *dix) dixMap(values map[group][]reflect.Type, data interface{}) error {
	val := reflect.ValueOf(data)

	if val.Type().Key().Kind() != reflect.String {
		return xerror.New("the map key should be string type")
	}

	next := val.MapRange()
	for next.Next() {
		var kind = next.Value().Type().Kind()
		if kind != reflect.Ptr && kind != reflect.Interface {
			return xerror.New("the map value should be Ptr or Interface type")
		}

		k := next.Key().String()
		if k == "" {
			return xerror.New("map key is null")
		}

		if x.isNil(next.Value()) {
			return xerror.Fmt("map value is nil, key:%s", k)
		}

		var tye = getIndirectType(next.Value().Type())

		if kind == reflect.Ptr {
			if ttk := x.getAbcType(tye); ttk != nil {
				x.setAbcValue(ttk, k, tye)
			}
		} else {
			x.setAbcValue(tye, k, tye)
		}

		x.setValue(tye, k, next.Value())
		values[k] = append(values[k], next.Value().Type())
	}

	return nil
}

func (x *dix) dixStruct(values map[group][]reflect.Type, data interface{}) error {
	val := reflect.ValueOf(data)
	tye := val.Type()

	for i := 0; i < tye.NumField(); i++ {
		kind := tye.Field(i).Type.Kind()
		if kind != reflect.Ptr && kind != reflect.Interface {
			return xerror.New("the struct field should be Ptr or Interface type")
		}

		if x.isNil(val.Field(i)) {
			return xerror.New("struct field data is nil")
		}

		ty := getIndirectType(tye.Field(i).Type)

		if kind == reflect.Ptr {
			if ttk := x.getAbcType(ty); ttk != nil {
				x.setAbcValue(ttk, x.getNS(tye.Field(i)), ty)
			}
		} else {
			x.setAbcValue(ty, x.getNS(tye.Field(i)), ty)
		}

		x.setValue(ty, x.getNS(tye.Field(i)), val.Field(i))

		values[x.getNS(tye.Field(i))] = append(values[x.getNS(tye.Field(i))], val.Field(i).Type())
	}

	return nil
}

func (x *dix) dixInterface(values map[group][]reflect.Type, val reflect.Value, name ...string) error {
	tye := getIndirectType(val.Type())
	var ns = _default
	if len(name) > 0 && name[0] != "" {
		ns = name[0]
	}

	x.setAbcValue(tye, ns, tye)
	x.setValue(tye, ns, val)
	values[ns] = append(values[ns], val.Type())
	return nil
}

func (x *dix) dixPtr(values map[group][]reflect.Type, val reflect.Value, name ...string) error {
	var ns = _default
	if len(name) > 0 && name[0] != "" {
		ns = name[0]
	}

	tye := getIndirectType(val.Type())
	if abcTy := x.getAbcType(tye); abcTy != nil {
		x.setAbcValue(abcTy, ns, tye)
	}

	x.setValue(tye, ns, val)
	values[ns] = append(values[ns], val.Type())
	return nil
}

func (x *dix) dixInterfaceInvoke(val reflect.Value, ns string) (err error) {
	defer xerror.RespErr(&err)
	tye := getIndirectType(val.Type())
	var vv = x.getAbcValue(tye, ns)
	xerror.Assert(!vv.IsValid() || vv.IsNil(), "namespace: %s not found", ns)
	val.Set(vv)
	return nil
}

func (x *dix) dixPtrInvoke(val reflect.Value, ns string) (err error) {
	defer xerror.RespErr(&err)
	tye := getIndirectType(val.Type())
	var vv = x.getValue(tye, ns)
	xerror.Assert(!vv.IsValid() || vv.IsNil(), "namespace: %s not found", ns)
	val.Set(vv)
	return nil
}

func (x *dix) dixStructInvoke(val reflect.Value) (err error) {
	defer xerror.RespErr(&err)

	tye := val.Type()
	for i := 0; i < tye.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() {
			continue
		}

		var kind = field.Type().Kind()

		// 结构体tag:dix, 类型为interface,ptr,struct
		if !x.hasNS(tye.Field(i)) ||
			kind != reflect.Ptr && kind != reflect.Interface && kind != reflect.Struct {
			continue
		}

		if kind == reflect.Struct {
			return x.dixStructInvoke(field)
		}

		var ns = x.getWithVal(tye.Field(i), val.Interface())

		var retVal reflect.Value
		if kind == reflect.Ptr {
			retVal = x.getValue(getIndirectType(tye.Field(i).Type), ns)
		} else {
			retVal = x.getAbcValue(getIndirectType(tye.Field(i).Type), ns)
		}

		xerror.Assert(!retVal.IsValid() || retVal.IsNil(), "value is nil, namespace:%s, field:%s", ns, tye.Field(i).Name)
		field.Set(retVal)
	}

	return nil
}
