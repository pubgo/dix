package dix_opts

import "math/rand"

type Option func(c *Options)
type Options struct {
	// 只要满足一个条件就可以
	OneIsOk bool
	// 允许nil值
	NilAllowed bool
	Strict     bool
	Rand       *rand.Rand
}
