package dix_internal

import (
	"reflect"

	"github.com/pubgo/funk/log"
)

const (
	// defaultKey 默认的 namespace
	defaultKey = "default"

	// InjectMethodPrefix 可以对对象进行 Inject, 只要这个对象的方法中包含了以`InjectMethodPrefix`为前缀的方法
	InjectMethodPrefix = "DixInject"
)

type (
	group      = string
	outputType = reflect.Type
	value      = reflect.Value
)

type Graph struct {
	Objects   string `json:"objects"`
	Providers string `json:"providers"`
}

var logger = log.GetLogger("dix")

func SetLog(setter func(logger log.Logger) log.Logger) {
	logger = setter(logger)
}
