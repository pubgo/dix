package dix

import "github.com/pubgo/xerror"

var defaultDix = New()

// Init ...
func Init(opts ...Option) error { return defaultDix.Init(opts...) }

// Dix ...
func Dix(data ...interface{}) error { return defaultDix.Dix(data...) }

// Go the dix must be ok
func Go(data ...interface{}) { xerror.Exit(defaultDix.Dix(data...)) }

// Graph dix graph
func Graph() string { return defaultDix.graph() }

func Start() error { return defaultDix.Start() }

func Stop() error { return defaultDix.stop() }
