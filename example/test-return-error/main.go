package main

import (
	"log"

	"github.com/pubgo/dix"
	"github.com/pubgo/funk/recovery"
)

func main() {
	defer recovery.Exit()

	di := dix.New(dix.WithValuesNull())
	di.Provide(func() (*log.Logger, error) {
		log.Println("provider ok")
		return new(log.Logger), nil
	})

	di.Inject(func(l *log.Logger) error {
		log.Println("inject ok")
		return nil
	})
}
