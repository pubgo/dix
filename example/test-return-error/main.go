package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/pubgo/dix"
	logger "github.com/pubgo/funk/log"
	"github.com/pubgo/funk/recovery"
	"github.com/pubgo/funk/try"
	"github.com/pubgo/funk/v2/result/resultchecker"
)

func testok() {
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

func testInjectErr() {
	di := dix.New(dix.WithValuesNull())
	di.Provide(func() (*log.Logger, error) {
		log.Println("provider ok")
		return new(log.Logger), nil
	})

	err := try.Try(func() error {
		di.Inject(func(l *log.Logger) error {
			return fmt.Errorf("inject_err")
		})
		return nil
	})
	if err != nil && strings.Contains(err.Error(), "inject_err") {
		return
	} else {
		panic(err)
	}
}

func testProviderErr() {
	di := dix.New(dix.WithValuesNull())
	di.Provide(func() (*log.Logger, error) {
		return nil, fmt.Errorf("provider_err")
	})

	err := try.Try(func() error {
		di.Inject(func(l *log.Logger) error {
			log.Println("inject ok")
			return nil
		})
		return nil
	})

	if err != nil && strings.Contains(err.Error(), "provider_err") {
		return
	} else {
		panic(err)
	}
}

func main() {
	defer recovery.Exit()

	resultchecker.RegisterErrCheck(logger.RecordErr())

	testok()
	testProviderErr()
	testInjectErr()
}
