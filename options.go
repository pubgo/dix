package dix

import "math/rand"

func WithRand(r *rand.Rand) Option {
	return func(c *Options) {
		c.rand = r
	}
}

func WithInvoker(invoker invokerFn) Option {
	return func(c *Options) {
		c.invokerFn = invoker
	}
}

func WithAllowNil(nilValueAllowed bool) Option {
	return func(c *Options) {
		c.nilValueAllowed = nilValueAllowed
	}
}

func WithStrict(strict bool) Option {
	return func(c *Options) {
		c.Strict = strict
	}
}
