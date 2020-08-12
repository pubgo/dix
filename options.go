package dix

import "math/rand"

func WithRand(r *rand.Rand) Option {
	return func(c *dix) {
		c.rand = r
	}
}

func WithInvoker(invoker invokerFn) Option {
	return func(c *dix) {
		c.invokerFn = invoker
	}
}

func WithAllowNil(nilValueAllowed bool) Option {
	return func(c *dix) {
		c.nilValueAllowed = nilValueAllowed
	}
}
