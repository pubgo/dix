package dix

import (
	"github.com/pubgo/dix/dix_opts"
)

var _dix = New()

// Init ...
func Init(opts ...dix_opts.Option) error { return _dix.Init(opts...) }

// Dix ...
// Deprecated: use Provider instead
func Dix(data ...interface{}) error      { return _dix.Dix(data...) }
func Provider(data ...interface{}) error { return _dix.Dix(data...) }

// Invoke 获取对象
// ns: namespace
func Invoke(data interface{}, ns ...string) error { return _dix.Invoke(data, ns...) }

// Graph dix graph
func Graph() string { return _dix.graph() }

// Json dix json graph
func Json() map[string]interface{} { return _dix.json() }

// New dix new
func New(opts ...dix_opts.Option) *dix { return newDix(opts...) }

type Go struct{}

func Start() error { return Provider(&Go{}) }
