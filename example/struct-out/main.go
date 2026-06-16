// Struct multi-output: one provider returns many dependencies via an Out struct.
//
// Run:
//
//	cd example/struct-out && go run .
package main

import (
	"fmt"

	"github.com/pubgo/dix/v2"
)

type Config struct {
	DSN string
}

type Database struct {
	Config *Config
}

type UserService struct {
	DB *Database
}

type OrderService struct {
	DB *Database
}

// In groups provider inputs; Out groups provider outputs.
type In struct {
	Config *Config
}

type Out struct {
	DB       *Database
	UserSvc  *UserService
	OrderSvc *OrderService
}

func main() {
	di := dix.New()

	dix.Provide(di, func() *Config {
		return &Config{DSN: "postgres://localhost/shop"}
	})

	dix.Provide(di, func(in In) Out {
		db := &Database{Config: in.Config}
		return Out{
			DB:       db,
			UserSvc:  &UserService{DB: db},
			OrderSvc: &OrderService{DB: db},
		}
	})

	dix.Inject(di, func(user *UserService, order *OrderService) {
		fmt.Println("user service dsn:", user.DB.Config.DSN)
		fmt.Println("order service dsn:", order.DB.Config.DSN)
	})
}
