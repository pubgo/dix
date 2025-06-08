package main

import (
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/recovery"
)

// 基础依赖类型
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

// 返回结构体，其字段需要依赖注入
type AppConfig struct {
	Logger   Logger
	Database Database
}

// Provider 返回 AppConfig 结构体
func createAppConfig() AppConfig {
	fmt.Println("创建 AppConfig")
	return AppConfig{
		// Logger 和 Database 字段需要从依赖中注入
	}
}

// 一个需要 AppConfig 的服务
type WebServer struct {
	Config AppConfig
}

func createWebServer(config AppConfig) *WebServer {
	fmt.Println("创建 WebServer")
	return &WebServer{Config: config}
}

func main() {
	defer recovery.Exit()

	fmt.Println("=== 测试输出结构体字段依赖解析 ===")

	// 注册基础依赖
	dixglobal.Provide(func() Logger {
		fmt.Println("创建 Logger")
		return &ConsoleLogger{}
	})

	dixglobal.Provide(func() Database {
		fmt.Println("创建 Database")
		return &MySQL{}
	})

	// 注册返回结构体的 provider，其字段需要依赖注入
	dixglobal.Provide(createAppConfig)

	// 注册一个使用 AppConfig 的服务
	dixglobal.Provide(createWebServer)

	fmt.Println("\n=== 依赖图 ===")
	graph := dixglobal.Graph()
	fmt.Printf("Providers:\n%s\n", graph.Providers)

	fmt.Println("\n=== 测试注入 ===")
	dixglobal.Inject(func(server *WebServer) {
		fmt.Println("获取到 WebServer")

		if server.Config.Logger != nil {
			server.Config.Logger.Log("Logger 注入成功")
		} else {
			fmt.Println("Logger 注入失败")
		}

		if server.Config.Database != nil {
			result := server.Config.Database.Query("SELECT * FROM users")
			fmt.Printf("Database 查询结果: %s\n", result)
		} else {
			fmt.Println("Database 注入失败")
		}
	})

	fmt.Println("\n=== 最终依赖图 ===")
	finalGraph := dixglobal.Graph()
	fmt.Printf("Objects:\n%s\n", finalGraph.Objects)

	fmt.Println("\n=== 测试完成 ===")
}
