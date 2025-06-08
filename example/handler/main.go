package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/assert"
	"github.com/pubgo/funk/recovery"
)

type Redis struct {
	name string
}

type Handler struct {
	Cli  *Redis
	Cli1 map[string]*Redis
}

func main() {
	defer recovery.Exit()

	defer func() {
		fmt.Println("\n=== Final Dependency Graph ===")
		graph := dixglobal.Graph()
		fmt.Printf("Providers:\n%s\n", graph.Providers)
		fmt.Printf("Objects:\n%s\n", graph.Objects)
	}()

	fmt.Println("=== Registering Providers ===")

	// 注册Logger提供者
	dixglobal.Provide(func() *log.Logger {
		return log.New(os.Stderr, "example: ", log.LstdFlags|log.Lshortfile)
	})

	// 注册Redis提供者（使用结构体参数）
	dixglobal.Provide(func(p struct {
		L *log.Logger
	}) *Redis {
		p.L.Println("Initializing Redis instance")
		return &Redis{name: "default-redis"}
	})

	// 注册Redis映射提供者
	dixglobal.Provide(func(l *log.Logger) map[string]*Redis {
		l.Println("Initializing Redis map")
		return map[string]*Redis{
			"ns":      {name: "hello1"},
			"default": {name: "default-redis"},
		}
	})

	fmt.Println("\n=== Initial Dependency Graph ===")
	graph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", graph.Providers)

	fmt.Println("\n=== Function Injection ===")
	dixglobal.Inject(func(r *Redis, l *log.Logger, rr map[string]*Redis) {
		l.Println("Function injection - invoking redis")
		fmt.Println("Injected Redis name:", r.name)
		fmt.Printf("Injected Redis map with %d entries:\n", len(rr))
		for key, redis := range rr {
			fmt.Printf("  '%s': %s\n", key, redis.name)
		}
	})

	fmt.Println("\n=== Struct Injection ===")
	h := dixglobal.Inject(new(Handler))
	assert.If(h.Cli.name != "default-redis", "inject error")
	assert.If(h.Cli1["ns"].name != "hello1", "inject error")

	fmt.Println("Struct injection successful:")
	fmt.Printf("  Handler.Cli.name: %s\n", h.Cli.name)
	fmt.Printf("  Handler.Cli1 map with %d entries:\n", len(h.Cli1))
	for key, redis := range h.Cli1 {
		fmt.Printf("    '%s': %s\n", key, redis.name)
	}

	fmt.Println("\n=== Struct Parameter Injection ===")
	dixglobal.Inject(func(h Handler) {
		assert.If(h.Cli.name != "default-redis", "inject error")
		assert.If(h.Cli1["ns"].name != "hello1", "inject error")

		fmt.Println("Struct parameter injection successful:")
		fmt.Printf("  Handler.Cli.name: %s\n", h.Cli.name)
		fmt.Printf("  Handler.Cli1 map with %d entries:\n", len(h.Cli1))
		for key, redis := range h.Cli1 {
			fmt.Printf("    '%s': %s\n", key, redis.name)
		}
	})

	fmt.Println("\n=== 通过 Inject 获取依赖实例演示 ===")
	// 使用 Inject 方法获取依赖实例
	var redis *Redis
	var redisMap map[string]*Redis
	var logger *log.Logger
	dixglobal.Inject(func(r *Redis, rm map[string]*Redis, l *log.Logger) {
		redis = r
		redisMap = rm
		logger = l
	})
	fmt.Println("Redis名称:", redis.name)
	fmt.Printf("Redis映射包含 %d 个条目:\n", len(redisMap))
	for key, r := range redisMap {
		fmt.Printf("  '%s': %s\n", key, r.name)
	}
	logger.Println("依赖获取演示完成")
}
