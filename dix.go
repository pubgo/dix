package dix

import (
	_ "embed"
	"log/slog"
	"reflect"

	"github.com/pubgo/dix/v2/dixinternal"
)

//go:embed .version
var version string

func ReleaseVersion() string { return version }

type (
	Option  = dixinternal.Option
	Options = dixinternal.Options
	Dix     = dixinternal.Dix
	Graph   = dixinternal.Graph
	// GraphOptions holds configuration options for graph rendering
	GraphOptions = dixinternal.GraphOptions
)

func SetLog(log slog.Handler) { dixinternal.SetLog(log) }

func WithValuesNull() Option { return dixinternal.WithValuesNull() }

func New(opts ...Option) *Dix { return dixinternal.New(opts...) }

// NewGraphOptions creates GraphOptions with sensible defaults
func NewGraphOptions() *GraphOptions { return dixinternal.NewGraphOptions() }

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
