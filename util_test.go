package dix

import (
	"fmt"
	"reflect"
	"testing"
)

func TestMakeMap(t *testing.T) {
	fmt.Println(MakeMap(map[string]reflect.Value{
		"hello": reflect.ValueOf("sss"),
	}).Interface())
}
