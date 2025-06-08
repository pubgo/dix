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

type ConsoleLogger struct{}

func (l *ConsoleLogger) Log(msg string) {
	fmt.Printf("[LOG] %s\n", msg)
}

type Database interface {
	Query(sql string) []string
}

type MockDatabase struct{}

func (db *MockDatabase) Query(sql string) []string {
	return []string{fmt.Sprintf("Result: %s", sql)}
}

// 嵌套结构体定义
type DatabaseConfig struct {
	Logger   Logger   // 可注入的字段
	Database Database // 可注入的字段
}

type ServiceConfig struct {
	DBConfig DatabaseConfig // 嵌套结构体
	Logger   Logger         // 可注入的字段
}

type UserService struct{}

func main() {
	defer recovery.Exit()

	fmt.Println("=== 嵌套结构体字段注入测试 ===")

	container := dix.New()

	// 注册基础依赖
	dix.Provide(container, func() Logger {
		return &ConsoleLogger{}
	})

	dix.Provide(container, func() Database {
		return &MockDatabase{}
	})

	// 测试: 带有嵌套结构体参数的provider
	fmt.Println("\n测试带嵌套结构体参数的provider:")
	dix.Provide(container, func(config ServiceConfig) *UserService {
		fmt.Printf("服务配置: %+v\n", config)
		fmt.Printf("数据库配置: %+v\n", config.DBConfig)

		// 验证顶层字段是否被正确注入
		if config.Logger != nil {
			config.Logger.Log("ServiceConfig Logger 工作正常")
		}

		// 验证嵌套结构体字段是否被正确注入
		if config.DBConfig.Logger != nil {
			config.DBConfig.Logger.Log("DatabaseConfig Logger 工作正常")
		}

		if config.DBConfig.Database != nil {
			results := config.DBConfig.Database.Query("SELECT * FROM users")
			fmt.Printf("嵌套数据库查询结果: %v\n", results)
		}

		return &UserService{}
	})

	// 注入并使用UserService
	_, err := dix.Inject(container, func(userService *UserService) {
		fmt.Printf("获取到用户服务: %+v\n", userService)
	})

	if err != nil {
		log.Printf("注入失败: %v", err)
		return
	}

	// 测试2: 直接注入嵌套结构体
	fmt.Println("\n测试直接注入嵌套结构体:")

	type AppHandler struct {
		Config ServiceConfig // 包含嵌套结构体的字段
	}

	handler := &AppHandler{}
	_, err = dix.Inject(container, handler)

	if err != nil {
		log.Printf("结构体注入失败: %v", err)
	} else {
		fmt.Printf("注入的处理器配置: %+v\n", handler.Config)
		fmt.Printf("嵌套的数据库配置: %+v\n", handler.Config.DBConfig)

		if handler.Config.Logger != nil {
			handler.Config.Logger.Log("Handler ServiceConfig Logger 工作正常")
		}

		if handler.Config.DBConfig.Logger != nil {
			handler.Config.DBConfig.Logger.Log("Handler DatabaseConfig Logger 工作正常")
		}

		if handler.Config.DBConfig.Database != nil {
			results := handler.Config.DBConfig.Database.Query("SELECT * FROM handlers")
			fmt.Printf("Handler 数据库查询: %v\n", results)
		}
	}

	fmt.Println("\n=== 测试完成 ===")
}
