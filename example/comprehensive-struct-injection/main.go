package main

import (
	"fmt"
	"log"

	"github.com/pubgo/dix"
	"github.com/pubgo/funk/recovery"
)

// 基础服务接口和实现
type Logger interface {
	Log(msg string)
}

type ConsoleLogger struct {
	Name string
}

func (l *ConsoleLogger) Log(msg string) {
	fmt.Printf("[%s] %s\n", l.Name, msg)
}

type Database interface {
	Query(sql string) []string
}

type MockDatabase struct {
	Name string
}

func (db *MockDatabase) Query(sql string) []string {
	return []string{fmt.Sprintf("Result from %s: %s", db.Name, sql)}
}

type Cache interface {
	Get(key string) string
	Set(key, value string)
}

type MemoryCache struct {
	Name string
	data map[string]string
}

func (c *MemoryCache) Get(key string) string {
	if c.data == nil {
		c.data = make(map[string]string)
	}
	return c.data[key]
}

func (c *MemoryCache) Set(key, value string) {
	if c.data == nil {
		c.data = make(map[string]string)
	}
	c.data[key] = value
}

// 复杂的嵌套结构体定义
type DatabaseConfig struct {
	Logger   Logger   // 可注入的接口字段
	Database Database // 可注入的接口字段
	Cache    Cache    // 可注入的接口字段
}

type ServiceConfig struct {
	DBConfig  DatabaseConfig    // 嵌套结构体
	Logger    Logger            // 可注入的接口字段
	Caches    []Cache           // 可注入的切片字段
	LoggerMap map[string]Logger // 可注入的映射字段
}

// 更深层的嵌套
type AppConfig struct {
	ServiceConfig ServiceConfig // 二级嵌套
	MainLogger    Logger        // 顶层可注入字段
}

type UserService struct {
	Name string
}

func main() {
	defer recovery.Exit()

	fmt.Println("=== 全面结构体字段注入测试 ===")

	container := dix.New()

	// 注册基础依赖
	dix.Provide(container, func() Logger {
		return &ConsoleLogger{Name: "MainLogger"}
	})

	dix.Provide(container, func() Database {
		return &MockDatabase{Name: "MainDB"}
	})

	dix.Provide(container, func() Cache {
		return &MemoryCache{Name: "MainCache"}
	})

	// 测试1: 简单结构体字段注入
	fmt.Println("\n1. 测试简单结构体字段注入:")
	dix.Provide(container, func(config DatabaseConfig) *UserService {
		fmt.Printf("数据库配置: %+v\n", config)

		if config.Logger != nil {
			config.Logger.Log("DatabaseConfig Logger 工作正常")
		}

		if config.Database != nil {
			results := config.Database.Query("SELECT * FROM users")
			fmt.Printf("数据库查询结果: %v\n", results)
		}

		if config.Cache != nil {
			config.Cache.Set("test", "value")
			value := config.Cache.Get("test")
			fmt.Printf("缓存测试结果: %s\n", value)
		}

		return &UserService{Name: "SimpleService"}
	})

	// 注入并使用UserService
	_, err := dix.Inject(container, func(userService *UserService) {
		fmt.Printf("获取到用户服务: %+v\n", userService)
	})

	if err != nil {
		log.Printf("简单注入失败: %v", err)
		return
	}

	// 测试2: 嵌套结构体字段注入
	fmt.Println("\n2. 测试嵌套结构体字段注入:")

	type NestedService struct {
		Name string
	}

	dix.Provide(container, func(config ServiceConfig) *NestedService {
		fmt.Printf("服务配置: %+v\n", config)
		fmt.Printf("嵌套数据库配置: %+v\n", config.DBConfig)

		// 验证顶层字段
		if config.Logger != nil {
			config.Logger.Log("ServiceConfig Logger 工作正常")
		}

		// 验证嵌套字段
		if config.DBConfig.Logger != nil {
			config.DBConfig.Logger.Log("嵌套 DatabaseConfig Logger 工作正常")
		}

		if config.DBConfig.Database != nil {
			results := config.DBConfig.Database.Query("SELECT * FROM nested")
			fmt.Printf("嵌套数据库查询: %v\n", results)
		}

		// 验证切片字段
		fmt.Printf("缓存切片长度: %d\n", len(config.Caches))
		for i, cache := range config.Caches {
			if cache != nil {
				cache.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
				fmt.Printf("缓存[%d]测试: %s\n", i, cache.Get(fmt.Sprintf("key%d", i)))
			}
		}

		// 验证映射字段
		fmt.Printf("Logger映射长度: %d\n", len(config.LoggerMap))
		for key, logger := range config.LoggerMap {
			if logger != nil {
				logger.Log(fmt.Sprintf("映射Logger[%s]工作正常", key))
			}
		}

		return &NestedService{Name: "NestedService"}
	})

	_, err = dix.Inject(container, func(nestedService *NestedService) {
		fmt.Printf("获取到嵌套服务: %+v\n", nestedService)
	})

	if err != nil {
		log.Printf("嵌套注入失败: %v", err)
		return
	}

	// 测试3: 深层嵌套结构体注入
	fmt.Println("\n3. 测试深层嵌套结构体注入:")

	type DeepService struct {
		Name string
	}

	dix.Provide(container, func(config AppConfig) *DeepService {
		fmt.Printf("应用配置: %+v\n", config)

		// 验证顶层字段
		if config.MainLogger != nil {
			config.MainLogger.Log("AppConfig MainLogger 工作正常")
		}

		// 验证二级嵌套字段
		if config.ServiceConfig.Logger != nil {
			config.ServiceConfig.Logger.Log("二级嵌套 ServiceConfig Logger 工作正常")
		}

		// 验证三级嵌套字段
		if config.ServiceConfig.DBConfig.Logger != nil {
			config.ServiceConfig.DBConfig.Logger.Log("三级嵌套 DatabaseConfig Logger 工作正常")
		}

		if config.ServiceConfig.DBConfig.Database != nil {
			results := config.ServiceConfig.DBConfig.Database.Query("SELECT * FROM deep")
			fmt.Printf("深层数据库查询: %v\n", results)
		}

		return &DeepService{Name: "DeepService"}
	})

	_, err = dix.Inject(container, func(deepService *DeepService) {
		fmt.Printf("获取到深层服务: %+v\n", deepService)
	})

	if err != nil {
		log.Printf("深层注入失败: %v", err)
		return
	}

	// 测试4: 直接注入复杂结构体
	fmt.Println("\n4. 测试直接注入复杂结构体:")

	type ComplexHandler struct {
		Config AppConfig // 包含深层嵌套的字段
	}

	handler := &ComplexHandler{}
	_, err = dix.Inject(container, handler)

	if err != nil {
		log.Printf("复杂结构体注入失败: %v", err)
	} else {
		fmt.Printf("注入的复杂处理器配置层级验证:\n")

		if handler.Config.MainLogger != nil {
			handler.Config.MainLogger.Log("Handler AppConfig MainLogger 工作正常")
		}

		if handler.Config.ServiceConfig.Logger != nil {
			handler.Config.ServiceConfig.Logger.Log("Handler ServiceConfig Logger 工作正常")
		}

		if handler.Config.ServiceConfig.DBConfig.Database != nil {
			results := handler.Config.ServiceConfig.DBConfig.Database.Query("SELECT * FROM handler")
			fmt.Printf("Handler 深层数据库查询: %v\n", results)
		}

		fmt.Printf("Handler 缓存切片长度: %d\n", len(handler.Config.ServiceConfig.Caches))
		fmt.Printf("Handler Logger映射长度: %d\n", len(handler.Config.ServiceConfig.LoggerMap))
	}

	fmt.Println("\n=== 全面测试完成 ===")
}
