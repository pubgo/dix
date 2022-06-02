package dix

import "github.com/pubgo/dix/internal/assert"

type Option func(opts *Options)
type Options struct {
	tagName string
}

func (o Options) Check() {
	assert.Msg(o.tagName == "", "tag name is null")
}

func WithTag(name string) Option {
	return func(opts *Options) {
		opts.tagName = name
	}
}
