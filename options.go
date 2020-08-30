package dix

import "math/rand"

func WithRand(r *rand.Rand) Option {
	return func(c *Options) {
		c.Rand = r
	}
}

func WithInvoker(invoker invokerFn) Option {
	return func(c *Options) {
		c.InvokerFn = invoker
	}
}

func WithAllowNil(nilValueAllowed bool) Option {
	return func(c *Options) {
		c.NilValueAllowed = nilValueAllowed
	}
}

func WithStrict(strict bool) Option {
	return func(c *Options) {
		c.Strict = strict
	}
}
