package dix_run

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/pubgo/dix"
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
	fmt.Println(dix.Graph())

	dt,_:=json.MarshalIndent(dix.Json()," ","")
	fmt.Println(string(dt))
}

func TestData(t *testing.T) {
	type testData struct {
		dix.Model
	}

	xerror.Exit(dix.Dix(func(*testData) {
		fmt.Println(time.Now())
	}))

	for i := 0; i < 5; i++ {
		xerror.Exit(dix.Dix(testData{}))
	}
}
