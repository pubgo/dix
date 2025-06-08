package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/recovery"
)

// 定义接口
type Logger interface {
	Log(msg string)
}

type Database interface {
	Query(sql string) []string
}

// 具体实现
type ConsoleLogger struct{}

func (l ConsoleLogger) Log(msg string) {
	fmt.Printf("[LOG] %s\n", msg)
}

type MockDatabase struct {
	Host string
}

func (db MockDatabase) Query(sql string) []string {
	return []string{"mock result"}
}

// 配置结构体
type AppConfig struct {
	Logger Logger
	DB     Database
	Name   string
}

// Provider 函数 - 返回结构体，应该能提供多种类型
func createAppConfig() AppConfig {
	fmt.Println("🔥 调用 createAppConfig")
	return AppConfig{
		Logger: ConsoleLogger{},
		DB:     MockDatabase{Host: "localhost"},
		Name:   "test-app",
	}
}

func main() {
	defer recovery.Exit()

	fmt.Println("=== 测试：多类型 Provider 设计 ===")

	// 注册 provider
	dixglobal.Provide(createAppConfig)

	fmt.Println("\n=== 依赖图 ===")
	graph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", graph.Providers)

	fmt.Println("\n=== 测试1：请求主要类型 AppConfig ===")
	dixglobal.Inject(func(config AppConfig) {
		fmt.Printf("✅ 获取到 AppConfig: Name=%s\n", config.Name)
		config.Logger.Log("配置加载成功")
	})

	fmt.Println("\n=== 测试2：请求字段类型 Logger ===")
	dixglobal.Inject(func(logger Logger) {
		fmt.Printf("✅ 获取到 Logger: %T\n", logger)
		logger.Log("直接注入的 Logger")
	})

	fmt.Println("\n=== 测试3：请求字段类型 Database ===")
	dixglobal.Inject(func(db Database) {
		fmt.Printf("✅ 获取到 Database: %T\n", db)
		result := db.Query("SELECT * FROM test")
		fmt.Printf("查询结果: %v\n", result)
	})

	fmt.Println("\n=== 测试完成 ===")
}
