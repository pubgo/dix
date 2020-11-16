package dix

import (
	"fmt"
	"testing"
	"time"

	"github.com/pubgo/xerror"
)

func TestStart(t *testing.T) {
	xerror.Exit(WithStart(func(ctx *StartCtx) {
		fmt.Println("start", time.Now())
	}))

	xerror.Exit(WithStop(func(ctx *StopCtx) {
		fmt.Println("stop", time.Now())
	}))

	for i := 0; i < 5; i++ {
		xerror.Exit(Start())
		xerror.Exit(Stop())
	}
	fmt.Println(Graph())
}

func TestData(t *testing.T) {
	type testData struct {
		Model
	}

	xerror.Exit(Dix(func(*testData) {
		fmt.Println(time.Now())
	}))

	for i := 0; i < 5; i++ {
		xerror.Exit(Dix(testData{}))
	}
}
