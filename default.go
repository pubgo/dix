package dix

import (
	"github.com/pubgo/dix/dix_opts"
)

var _dix = New()

// Init ...
func Init(opts ...dix_opts.Option) error { return _dix.Init(opts...) }

// Dix ...
// Deprecated: use Provider instead
func Dix(data ...interface{}) error { return _dix.Dix(data...) }

// Provider ...
func Provider(data ...interface{}) error { return _dix.Provider(data...) }

// Invoke 注入对象
// ns: namespace
// Deprecated: use Inject instead
func Invoke(data interface{}, ns ...string) error { return _dix.Invoke(data, ns...) }

// Inject 注入对象
// ns: namespace
func Inject(data interface{}, ns ...string) error { return _dix.Inject(data, ns...) }

// Graph dix graph
func Graph() string { return _dix.graph() }

// Json dix json graph
func Json() map[string]interface{} { return _dix.json() }

// New dix new
func New(opts ...dix_opts.Option) *dix { return newDix(opts...) }

type Resource struct{}

func FireResource() error { return _dix.Dix(&Resource{}) }
func ProviderResource(data ...func(_ *Resource) (interface{}, error)) error {
	var dataList []interface{}
	for i := range data {
		dataList = append(dataList, data[i])
	}
	return _dix.Dix(dataList...)
}
