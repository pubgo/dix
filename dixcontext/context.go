package dixcontext

import (
	"context"
	"log"

	"github.com/pubgo/dix/v2"
)

type dixKey struct{}

func Get(ctx context.Context) *dix.Dix {
	if ctx == nil {
		log.Panicln("ctx is nil")
	}

	di, ok := ctx.Value(dixKey{}).(*dix.Dix)
	if !ok {
		log.Panicln("dix not found")
	}
	return di
}

func Create(ctx context.Context, dix *dix.Dix) context.Context {
	if ctx == nil {
		log.Panicln("ctx is nil")
	}

	if dix == nil {
		log.Panicln("dix is nil")
	}

	return context.WithValue(ctx, dixKey{}, dix)
}
