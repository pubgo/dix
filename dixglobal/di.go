package dixglobal

import (
	"reflect"

	"github.com/pubgo/dix/dixinternal"
)

var _dix = dixinternal.New(dixinternal.WithValuesNull())

// Example:
//
//	c := di.New()
//	c.Provide(func() *Config { return &Config{Endpoint: "localhost:..."} }) // Configuration
//	c.Provide(NewDB)                                                  // Database connection
//	c.Provide(NewHTTPServer)                                          // Server
//
//	c.Invoke(func(server *http.Server) { // Application startup
//		server.ListenAndServe()
//	})
//
// For more usage details, see the documentation for the Container type.

// Provide registers an object constructor
func Provide(data any) {
	_dix.Provide(data)
}

// Inject injects objects
//
//	data: <*struct> or <func>
func Inject[T any](data T, opts ...dixinternal.Option) T {
	vp := reflect.ValueOf(data)
	if vp.Kind() == reflect.Struct {
		_ = _dix.Inject(&data, opts...)
	} else {
		_ = _dix.Inject(data, opts...)
	}
	return data
}

// Graph Dix graph
func Graph() *dixinternal.Graph {
	return _dix.Graph()
}
