package dix

import "github.com/pubgo/dix/internal/assert"

type Option func(*Options)
type Options struct {
	tagName string
}

func (o Options) Check() {
	assert.Msg(o.tagName == "", "tag name is null")
}
