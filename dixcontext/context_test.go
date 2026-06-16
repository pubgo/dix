package dixcontext

import (
	"context"
	"testing"

	"github.com/pubgo/dix/v2"
)

func TestCreateAndGet(t *testing.T) {
	di := dix.New()
	ctx := Create(context.Background(), di)

	got := Get(ctx)
	if got != di {
		t.Fatal("Get returned unexpected container")
	}
}

func TestGetOrNil(t *testing.T) {
	if got := GetOrNil(nil); got != nil {
		t.Fatal("GetOrNil(nil) should return nil")
	}

	if got := GetOrNil(context.Background()); got != nil {
		t.Fatal("GetOrNil without container should return nil")
	}
}

func TestCreatePanicsOnNilInput(t *testing.T) {
	di := dix.New()

	assertPanic(t, func() {
		_ = Create(nil, di)
	})

	assertPanic(t, func() {
		_ = Create(context.Background(), nil)
	})
}

func TestGetPanicsOnNilOrMissing(t *testing.T) {
	assertPanic(t, func() {
		_ = Get(nil)
	})

	assertPanic(t, func() {
		_ = Get(context.Background())
	})
}

func assertPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}
