package dix_inter

import "reflect"

const (
	defaultKey         = "default"
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
