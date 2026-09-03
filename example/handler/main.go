// 【功能】常见装配模式:logger 单例 + 命名空间 map + 函数/结构体混合注入。
//
// 【原理】同类型依赖在容器内是单例:
//   - *log.Logger 只创建一次,两个 Redis provider 与注入函数共享同一实例;
//   - map[string]*Redis 提供命名空间版本,与单值 *Redis 并存互不干扰;
//   - 结构体注入时,指针字段取单值,map 字段取命名空间集合。
//
// 该语义由 dixinternal 的 TestPatternSingletonSharing 锁定。
//
// 【运行】
//
//	cd example/handler && go run .
//
// 【预期输出】(日志时间戳省略;另有一条 "provider value already exists" 提示日志)
//
//	app: init default redis
//	app: init namespaced redis map
//	app: invoke default redis: 127.0.0.1:6379
//	handler.Redis: 127.0.0.1:6379
//	handler.All[cache]: 127.0.0.1:6380
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pubgo/dix/v2"
)

type Redis struct {
	Addr string
}

type Handler struct {
	Logger *log.Logger
	Redis  *Redis
	All    map[string]*Redis
}

func main() {
	di := dix.New()

	// Logger 是单例:所有消费者拿到的都是同一个实例。
	dix.Provide(di, func() *log.Logger {
		return log.New(os.Stderr, "app: ", log.LstdFlags)
	})

	dix.Provide(di, func(l *log.Logger) *Redis {
		l.Println("init default redis")
		return &Redis{Addr: "127.0.0.1:6379"}
	})

	// 命名空间版本:与单值 *Redis 并存,互不覆盖。
	dix.Provide(di, func(l *log.Logger) map[string]*Redis {
		l.Println("init namespaced redis map")
		return map[string]*Redis{
			"cache": {Addr: "127.0.0.1:6380"},
		}
	})

	// 函数注入。
	dix.Inject(di, func(r *Redis, l *log.Logger) {
		l.Println("invoke default redis:", r.Addr)
	})

	// 结构体注入:Redis 字段取单值,All 字段取命名空间集合。
	h := &Handler{}
	dix.Inject(di, h)

	fmt.Println("handler.Redis:", h.Redis.Addr)
	fmt.Println("handler.All[cache]:", h.All["cache"].Addr)
}
