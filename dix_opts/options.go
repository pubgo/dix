package dix_opts

import "math/rand"

type Option func(c *Options)
type Options struct {
	// 允许nil值
	NilAllowed bool
	Rand       *rand.Rand
}
