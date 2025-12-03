package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/pubgo/dix/v2"
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
	var injectErr error
	func() { // Use a closure to capture panic as error
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					injectErr = err
				} else {
					injectErr = fmt.Errorf("panic: %v", r)
				}
			}
		}()
		di.Inject(func(l *log.Logger) error {
			return fmt.Errorf("inject_err")
		})
	}()

	if injectErr != nil {
		log.Printf("inject error occurred, inject_err: %v\n", injectErr)
	}

	if injectErr != nil && strings.Contains(injectErr.Error(), "inject_err") {
		return
	} else if injectErr != nil {
		panic(injectErr)
	}
}

func testProviderErr() {
	di := dix.New(dix.WithValuesNull())
	di.Provide(func() (*log.Logger, error) {
		return nil, fmt.Errorf("provider_err")
	})

	var provideErr error
	func() { // Use a closure to capture panic as error
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					provideErr = err
				} else {
					provideErr = fmt.Errorf("panic: %v", r)
				}
			}
		}()
		di.Inject(func(l *log.Logger) error {
			log.Println("inject ok")
			return nil
		})
	}()

	if provideErr != nil {
		log.Printf("provider error occurred, provider_err: %v\n", provideErr)
	}

	if provideErr != nil && strings.Contains(provideErr.Error(), "provider_err") {
		return
	} else if provideErr != nil {
		panic(provideErr)
	}
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic: %v\n", r)
		}
	}()

	testok()
	testProviderErr()
	testInjectErr()
}
