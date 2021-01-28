package dix

import (
	"reflect"

	"github.com/pubgo/xerror"
)

// dix(struct)
// dix(map)
// dix(ptr)

func (x *dix) handleStruct() {}
func (x *dix) handleMap()    {}
func (x *dix) handlePtr()    {}

func (x *dix) dixMap(values map[ns][]reflect.Type, data interface{}) error {
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

		if ttk := x.checkAbcImplement(indirectType(iter.Value().Type())); ttk != nil {
			x.setAbcValue(ttk, k, indirectType(iter.Value().Type()))
		}

		x.setValue(indirectType(iter.Value().Type()), k, iter.Value())
		values[k] = append(values[k], iter.Value().Type())
	}

	return nil
}

func (x *dix) dixStruct(values map[ns][]reflect.Type, data interface{}) error {
	val := reflect.ValueOf(data)
	tye := val.Type()

	for i := 0; i < tye.NumField(); i++ {
		if tye.Field(i).Type.Kind() != reflect.Ptr {
			return xerror.New("the struct field should be Ptr type")
		}

		if x.isNil(val.Field(i)) {
			return xerror.New("struct field data is nil")
		}

		if ttk := x.checkAbcImplement(indirectType(tye.Field(i).Type)); ttk != nil {
			x.setAbcValue(ttk, x.getNS(tye.Field(i)), indirectType(tye.Field(i).Type))
		}
		x.setValue(indirectType(tye.Field(i).Type), x.getNS(tye.Field(i)), val.Field(i))
		values[x.getNS(tye.Field(i))] = append(values[x.getNS(tye.Field(i))], val.Field(i).Type())
	}

	return nil
}

func (x *dix) dixPtr(values map[ns][]reflect.Type, data interface{}) error {
	val := reflect.ValueOf(data)
	if x.isNil(val) {
		return xerror.New("data is nil")
	}

	tye := indirectType(val.Type())
	if ttk := x.checkAbcImplement(tye); ttk != nil {
		x.setAbcValue(ttk, _default, tye)
	}

	x.setValue(tye, _default, val)
	values[_default] = append(values[_default], val.Type())
	return nil
}
