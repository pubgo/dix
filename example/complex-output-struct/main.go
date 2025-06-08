package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/recovery"
)

// 基础依赖
type Logger interface {
	Log(message string)
}

type Database interface {
	Query(sql string) string
}

type ConsoleLogger struct{}

func (c *ConsoleLogger) Log(message string) {
	fmt.Printf("[LOG] %s\n", message)
}

type MySQL struct{}

func (m *MySQL) Query(sql string) string {
	return fmt.Sprintf("[MySQL] %s", sql)
}

// 嵌套结构体
type DatabaseConfig struct {
	Host Database
}

// 主配置结构体
type AppConfig struct {
	Logger Logger
	DB     DatabaseConfig // 嵌套结构体
}

// Provider 1: 有初始化逻辑的provider
func createAppConfig() AppConfig {
	fmt.Println("🔥 调用 createAppConfig provider")
	return AppConfig{
		// Logger 和 DB 字段应该从依赖中注入
	}
}

// Provider 2: 显式创建DatabaseConfig
func createDatabaseConfig() DatabaseConfig {
	fmt.Println("🔥 调用 createDatabaseConfig provider")
	return DatabaseConfig{
		// Host 字段应该从依赖中注入
	}
}

// 测试服务
type WebServer struct {
	Config AppConfig
}

func createWebServer(config AppConfig) *WebServer {
	fmt.Println("🔥 调用 createWebServer provider")
	return &WebServer{Config: config}
}

func main() {
	defer recovery.Exit()

	fmt.Println("=== 复杂输出结构体字段依赖测试 ===")

	// 注册基础依赖
	dixglobal.Provide(func() Logger {
		fmt.Println("🔥 创建 Logger")
		return &ConsoleLogger{}
	})

	dixglobal.Provide(func() Database {
		fmt.Println("🔥 创建 Database")
		return &MySQL{}
	})

	// 注册结构体providers
	dixglobal.Provide(createDatabaseConfig)
	dixglobal.Provide(createAppConfig)
	dixglobal.Provide(createWebServer)

	fmt.Println("\n=== 依赖图 ===")
	graph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", graph.Providers)

	fmt.Println("\n=== 测试注入 ===")
	dixglobal.Inject(func(server *WebServer) {
		fmt.Println("✅ 获取到 WebServer")

		// 测试Logger
		if server.Config.Logger != nil {
			fmt.Println("✅ AppConfig.Logger 注入成功")
			server.Config.Logger.Log("Logger 测试")
		} else {
			fmt.Println("❌ AppConfig.Logger 注入失败")
		}

		// 测试嵌套结构体
		if server.Config.DB.Host != nil {
			fmt.Println("✅ AppConfig.DB.Host 注入成功")
			result := server.Config.DB.Host.Query("SELECT * FROM config")
			fmt.Printf("   查询结果: %s\n", result)
		} else {
			fmt.Println("❌ AppConfig.DB.Host 注入失败")
		}
	})

	fmt.Println("\n=== 最终依赖图 ===")
	finalGraph := dixglobal.Graph()
	fmt.Printf("Objects:\n%s\n", finalGraph.Objects)

	fmt.Println("\n=== 测试完成 ===")
}
