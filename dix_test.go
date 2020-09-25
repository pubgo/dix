package dix

import (
	"fmt"
	"testing"

	"github.com/pubgo/xerror"
)

func TestStart(t *testing.T) {
	Go(func(ctx *StartCtx) {
		fmt.Println("start", ctx.data)
	})

	Go(func(ctx *StopCtx) {
		fmt.Println("stop", ctx.data)
	})

	for i := 0; i < 5; i++ {
		xerror.Exit(Start())
		xerror.Exit(Stop())
	}
	fmt.Println(Graph())
}
