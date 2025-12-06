package dixinternal

import (
	"reflect"
)

const (
	// defaultKey default namespace
	defaultKey = "default"

	// InjectMethodPrefix can inject objects, as long as the method of this object contains a prefix of `InjectMethodPrefix`
	InjectMethodPrefix = "DixInject"
)

type (
	group      = string
	outputType = reflect.Type
	value      = reflect.Value
)

type Graph struct {
	Objects       string `json:"objects"`
	Providers     string `json:"providers"`
	ProviderTypes string `json:"provider_types"`
}
