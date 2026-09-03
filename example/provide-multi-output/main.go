// 【功能】结构体多输出:一个 provider 通过返回 Out 结构体,一次提供多个依赖。
//
// 【原理】provider 返回 struct 时,dix 按导出字段逐一拆分注册
// (仅注册指针/接口/函数/聚合类型的字段):
//   - 各字段共享同一底层实例:UserSvc.DB 与 OrderSvc.DB 是同一个 *Database 指针;
//   - provider 本身只执行一次,产物缓存后按需分发。
//
// 该语义由 dixinternal 的 TestPatternStructOutSharedInstance
// 与本目录 main_test.go 的 TestBuildServices 锁定。
//
// 【运行】
//
//	cd example/provide-multi-output && go run .
//
// 【预期输出】
//
//	user service dsn: postgres://localhost/shop
//	order service dsn: postgres://localhost/shop
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

// In 聚合 provider 的输入依赖;Out 聚合 provider 的输出依赖。
type In struct {
	Config *Config
}

type Out struct {
	DB       *Database
	UserSvc  *UserService
	OrderSvc *OrderService
}

func main() {
	user, order := buildServices()

	// user.DB 与 order.DB 是同一个 *Database 实例。
	fmt.Println("user service dsn:", user.DB.Config.DSN)
	fmt.Println("order service dsn:", order.DB.Config.DSN)
}

// buildServices 注册 Config 与多输出 provider,注入并返回两个服务。
func buildServices() (*UserService, *OrderService) {
	di := dix.New()

	dix.Provide(di, func() *Config {
		return &Config{DSN: "postgres://localhost/shop"}
	})

	// 返回 Out:dix 把 DB、UserSvc、OrderSvc 三个字段拆开分别注册。
	dix.Provide(di, func(in In) Out {
		db := &Database{Config: in.Config}
		return Out{
			DB:       db,
			UserSvc:  &UserService{DB: db},
			OrderSvc: &OrderService{DB: db},
		}
	})

	var user *UserService
	var order *OrderService
	dix.Inject(di, func(u *UserService, o *OrderService) { user, order = u, o })
	return user, order
}
