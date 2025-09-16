package dixcontext

import (
	"context"

	"github.com/pubgo/dix/v2"
)

const ctxKey = "__dix"

func Get(ctx context.Context) *dix.Dix {
	if ctx == nil {
		panic("ctx is nil")
	}

	di, ok := ctx.Value(ctxKey).(*dix.Dix)
	if !ok {
		panic("dix not found")
	}
	return di
}

func Create(ctx context.Context, dix *dix.Dix) context.Context {
	if ctx == nil {
		panic("ctx is nil")
	}

	if dix == nil {
		panic("dix is nil")
	}

	return context.WithValue(ctx, ctxKey, dix)
}
