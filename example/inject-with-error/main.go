package main

import (
	"errors"
	"fmt"

	"github.com/pubgo/dix/dixglobal"
	"github.com/pubgo/funk/recovery"
)

// 模拟一些服务接口
type Logger interface {
	Log(message string)
}

type Database interface {
	Connect() error
	Query(sql string) ([]string, error)
}

type ConsoleLogger struct{}

func (c *ConsoleLogger) Log(message string) {
	fmt.Printf("[LOG] %s\n", message)
}

type MockDatabase struct {
	connected bool
}

func (m *MockDatabase) Connect() error {
	m.connected = true
	fmt.Println("[DB] 数据库连接成功")
	return nil
}

func (m *MockDatabase) Query(sql string) ([]string, error) {
	if !m.connected {
		return nil, errors.New("数据库未连接")
	}
	return []string{"row1", "row2"}, nil
}

// 启动服务的函数，无返回值
func startService(logger Logger, db Database) {
	logger.Log("启动服务...")
	logger.Log("服务启动完成")
}

// 初始化数据库的函数，有 error 返回值
func initDatabase(logger Logger, db Database) error {
	logger.Log("初始化数据库...")

	err := db.Connect()
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 执行一些初始化查询
	_, err = db.Query("CREATE TABLE IF NOT EXISTS users (id INT, name TEXT)")
	if err != nil {
		return fmt.Errorf("创建表失败: %w", err)
	}

	logger.Log("数据库初始化完成")
	return nil
}

// 一个会返回错误的函数
func failingFunction(logger Logger) error {
	logger.Log("执行一个会失败的操作...")
	return errors.New("模拟错误：操作失败")
}

// 非 error 返回值的函数（应该被拒绝）
func invalidFunction(logger Logger) string {
	return "this should not be allowed"
}

func main() {
	defer recovery.Exit()

	fmt.Println("=== 测试 Inject 函数支持 error 返回值 ===")

	// 注册依赖
	dixglobal.Provide(func() Logger {
		return &ConsoleLogger{}
	})

	dixglobal.Provide(func() Database {
		return &MockDatabase{}
	})

	// 获取容器实例以便测试错误处理
	container := dixglobal.Container()

	fmt.Println("\n=== 测试1: 无返回值的函数注入 ===")
	err := container.Inject(startService)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Println("✅ 无返回值函数注入成功")
	}

	fmt.Println("\n=== 测试2: 有 error 返回值且成功的函数注入 ===")
	err = container.Inject(initDatabase)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Println("✅ 有 error 返回值的函数注入成功")
	}

	fmt.Println("\n=== 测试3: 有 error 返回值且失败的函数注入 ===")
	err = container.Inject(failingFunction)
	if err != nil {
		fmt.Printf("✅ 正确捕获了函数返回的错误: %v\n", err)
	} else {
		fmt.Println("❌ 应该捕获到错误但没有")
	}

	fmt.Println("\n=== 测试4: 尝试注入非 error 返回值的函数（应该失败）===")
	err = container.Inject(invalidFunction)
	if err != nil {
		fmt.Printf("✅ 正确拒绝了非 error 返回值的函数: %v\n", err)
	} else {
		fmt.Println("❌ 应该拒绝非 error 返回值的函数但没有")
	}

	fmt.Println("\n=== 测试5: 使用匿名函数（有 error 返回值）===")
	err = container.Inject(func(logger Logger, db Database) error {
		logger.Log("使用匿名函数执行操作...")

		// 模拟一些可能失败的操作
		_, err := db.Query("SELECT * FROM users")
		if err != nil {
			return fmt.Errorf("查询失败: %w", err)
		}

		logger.Log("匿名函数执行完成")
		return nil
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Println("✅ 匿名函数（有 error 返回值）注入成功")
	}

	fmt.Println("\n=== 测试6: 使用匿名函数（无返回值）===")
	err = container.Inject(func(logger Logger) {
		logger.Log("使用无返回值的匿名函数")
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Println("✅ 匿名函数（无返回值）注入成功")
	}

	fmt.Println("\n=== 测试7: 使用 dixglobal.Inject (会自动处理错误) ===")
	// 对于成功的情况，可以使用 dixglobal.Inject
	dixglobal.Inject(func(logger Logger) {
		logger.Log("使用 dixglobal.Inject 成功案例")
	})
	fmt.Println("✅ dixglobal.Inject 成功案例完成")

	fmt.Println("\n=== 测试完成 ===")
}
