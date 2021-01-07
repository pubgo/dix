package dix_opts

import "math/rand"

type Option func(c *Options)
type Options struct {
	NilValueAllowed bool
	Strict          bool
	Rand            *rand.Rand
}

func WithRand(r *rand.Rand) Option {
	return func(c *Options) {
		c.Rand = r
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
