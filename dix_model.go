package dix

import (
	"reflect"
	"time"
)

type Model struct{ Data int64 }

func (t Model) init() {}

type dixData interface{ init() }

// checkDixDataType
// 检查是否实现dixData
func checkDixDataType(data dixData) interface{} {
	dt := reflect.New(getIndirectType(reflect.TypeOf(data)))
	dt.Elem().FieldByName("Data").Set(reflect.ValueOf(time.Now().UnixNano()))
	return dt.Interface()
}
