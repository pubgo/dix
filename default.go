package dix

import (
	"github.com/pubgo/dix/dix_opts"
)

var defaultDix = New()

// Init ...
func Init(opts ...dix_opts.Option) error { return defaultDix.Init(opts...) }

// Dix ...
func Dix(data ...interface{}) error { return defaultDix.Dix(data...) }

// Invoke 获取对象
func Invoke(data interface{}, namespaces ...string) error {
	return defaultDix.Invoke(data, namespaces...)
}

// Graph dix graph
func Graph() string { return defaultDix.graph() }

// Json dix json graph
func Json() map[string]interface{} { return defaultDix.json() }

// New dix new
func New(opts ...dix_opts.Option) *dix { return newDix(opts...) }
