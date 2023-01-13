package dix

import "reflect"

const (
	defaultKey         = "default"
	InjectMethodPrefix = "DixInject"
)

type (
	group = string
	key   = reflect.Type
	value = reflect.Value
)

type Graph struct {
	Objects   string `json:"objects"`
	Providers string `json:"providers"`
}
