package dix

import (
	"fmt"
	"testing"
	"time"

	"github.com/pubgo/xerror"
)

func TestStart(t *testing.T) {
	xerror.Exit(WithStart(func() {
		fmt.Println("start", time.Now())
	}))

	xerror.Exit(WithStop(func() {
		fmt.Println("stop", time.Now())
	}))

	for i := 0; i < 5; i++ {
		xerror.Exit(Start())
		xerror.Exit(Stop())
	}
	fmt.Println(Graph())
}
