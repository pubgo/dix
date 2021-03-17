package dix

import (
	"github.com/pubgo/xerror"
	"reflect"
)

// dix(func(struct))
func (x *dix) handleStructFn(data interface{}) (gErr error) {
	defer xerror.Resp(func(err xerror.XErr) { gErr = err.WrapF("[dix] unknown error, data:%#v", data) })

	sVal := reflect.ValueOf(data)
	sTyp := sVal.Type()

	// 结构体类型检查
	xerror.Assert(sTyp.Kind() != reflect.Func, "[data] %s should be func type", sTyp.Name())
	xerror.Assert(x.isNil(sVal), "[dix] value is nil")

	typ := getIndirectType(sTyp)
	if ttk := x.getAbcType(typ); ttk != nil {
		x.setAbcValue(ttk, _default, typ)
	}

	x.setValue(typ, _default, sVal)
	values[_default] = append(values[_default], sTyp)

	return values, nil
}

// dix(func(ptr))
func (x *dix) handlePtrFn() {}
