package dix

import "reflect"

const (
	defaultKey = "default"
)

type (
	group = string
	key   = reflect.Type
	value = reflect.Value
)
