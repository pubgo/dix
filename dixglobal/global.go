package dixglobal

import (
	"reflect"

	"github.com/pubgo/dix/dixinternal"
	"github.com/pubgo/funk/assert"
)

var _container = dixinternal.New(dixinternal.WithValuesNull())

// Example:
//
//	Provide(func() *Config { return &Config{Endpoint: "localhost:..."} }) // Configuration
//	Provide(NewDB)                                                        // Database connection
//	Provide(NewHTTPServer)                                                // Server
//
//	Inject(func(server *http.Server) { // Application startup
//		server.ListenAndServe()
//	})
//
// For more usage details, see the documentation for the Container type.

// Provide registers an object constructor
func Provide(provider any) {
	assert.Must(_container.Provide(provider))
}

// Inject injects objects
//
//	target: <*struct> or <func>
func Inject[T any](target T, opts ...dixinternal.Option) T {
	vp := reflect.ValueOf(target)
	if vp.Kind() == reflect.Struct {
		assert.Must(_container.Inject(&target, opts...))
	} else {
		assert.Must(_container.Inject(target, opts...))
	}
	return target
}

// Get retrieves an instance of the specified type
func Get[T any](opts ...dixinternal.Option) T {
	result, err := dixinternal.Get[T](_container, opts...)
	assert.Must(err)
	return result
}

// MustGet retrieves an instance of the specified type, panics on error
func MustGet[T any](opts ...dixinternal.Option) T {
	return dixinternal.MustGet[T](_container, opts...)
}

// Graph returns the dependency graph
func Graph() *dixinternal.Graph {
	return _container.Graph()
}

// Container returns the global container instance
func Container() dixinternal.Container {
	return _container
}
