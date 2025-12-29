package dix

import (
	_ "embed"
	"log/slog"
	"reflect"

	"github.com/pubgo/dix/v2/dixinternal"
	"github.com/pubgo/dix/v2/dixrender"
)

//go:embed .release
var version string

func ReleaseVersion() string { return version }

type (
	Option  = dixinternal.Option
	Options = dixinternal.Options
	Dix     = dixinternal.Dix
	// Graph represents dependency graphs in DOT format
	Graph = dixrender.Graph
	// GraphOptions holds configuration options for graph rendering
	GraphOptions = dixrender.GraphOptions
)

func SetLog(log slog.Handler) { dixinternal.SetLog(log) }

func WithValuesNull() Option { return dixinternal.WithValuesNull() }

func New(opts ...Option) *Dix { return dixinternal.New(opts...) }

// NewGraphOptions creates GraphOptions with sensible defaults
func NewGraphOptions() *GraphOptions { return dixrender.NewGraphOptions() }

func Inject[T any](di *Dix, data T, opts ...Option) T {
	vp := reflect.ValueOf(data)
	if vp.Kind() == reflect.Struct {
		_ = di.Inject(&data, opts...)
	} else {
		_ = di.Inject(data, opts...)
	}

	return data
}

func InjectT[T any](di *Dix, opts ...Option) T {
	var data T
	if reflect.TypeOf(data).Kind() != reflect.Struct {
		panic("<T> type kind is not struct")
	}

	_ = di.Inject(&data, opts...)
	return data
}

func Provide(di *Dix, data any) { di.Provide(data) }

// GenerateGraph generates dependency graphs with default options
// This function uses dixrender module to generate graphs
func GenerateGraph(di *Dix) *Graph {
	return GenerateGraphWithOptions(di, dixrender.NewGraphOptions())
}

// GenerateGraphWithOptions generates dependency graphs with custom options
// This function uses dixrender module to generate graphs
func GenerateGraphWithOptions(di *Dix, opts *GraphOptions) *Graph {
	adapter := dixrender.NewDixAdapter(di)
	return dixrender.GenerateGraphWithOptions(adapter, opts)
}
