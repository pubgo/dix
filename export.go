package dix

import (
	"math/rand"
	"time"
)

func New(opts ...Option) *dix {
	c := &dix{
		providers:       make(map[key]map[ns][]*node),
		abcProviders:    make(map[key]map[ns][]*node),
		values:          make(map[key]map[ns]value),
		abcValues:       make(map[key]map[ns]key),
		rand:            rand.New(rand.NewSource(time.Now().UnixNano())),
		invokerFn:       defaultInvoker,
		nilValueAllowed: false,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (x *dix) Dix(data interface{}) error { return x.dix(data) }
func (x *dix) Graph() string              { return x.graph() }
