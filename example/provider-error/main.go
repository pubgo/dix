package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/pubgo/dix"
	"github.com/pubgo/funk/recovery"
)

type Database interface {
	Connect() error
	Query(sql string) ([]string, error)
}

type PostgresDB struct {
	host string
	port int
}

func (db *PostgresDB) Connect() error {
	fmt.Printf("Connecting to PostgreSQL at %s:%d\n", db.host, db.port)
	return nil
}

func (db *PostgresDB) Query(sql string) ([]string, error) {
	fmt.Printf("Executing query: %s\n", sql)
	return []string{"result1", "result2"}, nil
}

type Config struct {
	DatabaseHost string
	DatabasePort int
	EnableCache  bool
}

type UserService struct {
	DB     Database
	Config *Config
}

func main() {
	defer recovery.Exit()

	container := dix.New()

	fmt.Println("=== Provider with Error Return Demo ===")

	// 注册配置提供者（成功案例）
	dix.Provide(container, func() (*Config, error) {
		fmt.Println("Creating config...")
		config := &Config{
			DatabaseHost: "localhost",
			DatabasePort: 5432,
			EnableCache:  true,
		}
		return config, nil // 返回 nil error，表示成功
	})

	// 注册数据库提供者（依赖配置）
	dix.Provide(container, func(config *Config) (Database, error) {
		fmt.Printf("Creating database connection with config: %+v\n", config)

		if config.DatabaseHost == "" {
			return nil, errors.New("database host cannot be empty")
		}

		db := &PostgresDB{
			host: config.DatabaseHost,
			port: config.DatabasePort,
		}

		// 模拟连接测试
		if err := db.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}

		return db, nil // 成功返回
	})

	// 注册用户服务提供者
	dix.Provide(container, func(db Database, config *Config) (*UserService, error) {
		fmt.Println("Creating user service...")

		if !config.EnableCache {
			fmt.Println("Warning: Cache is disabled")
		}

		service := &UserService{
			DB:     db,
			Config: config,
		}

		return service, nil
	})

	fmt.Println("\n=== Successful Provider Chain ===")

	// 示例1：正常获取实例
	fmt.Println("=== 正常实例获取 ===")
	var userService *UserService
	err := dix.Inject(container, func(us *UserService) {
		userService = us
	})
	if err != nil {
		log.Printf("注入失败: %v", err)
		return
	}
	fmt.Printf("UserService 实例获取成功: %p\n", userService)

	// 测试数据库查询
	results, err := userService.DB.Query("SELECT * FROM users")
	if err != nil {
		fmt.Printf("Query error: %v\n", err)
	} else {
		fmt.Printf("Query results: %v\n", results)
	}

	fmt.Println("\n=== Error Provider Demo ===")

	// 创建新容器来演示错误情况
	errorContainer := dix.New()

	// 注册一个会返回错误的配置提供者
	dix.Provide(errorContainer, func() (*Config, error) {
		fmt.Println("Attempting to create config...")
		return nil, errors.New("configuration service is unavailable")
	})

	// 注册依赖配置的数据库提供者
	dix.Provide(errorContainer, func(config *Config) (Database, error) {
		// 这个不会被调用，因为配置提供者会失败
		fmt.Println("This should not be printed")
		return &PostgresDB{host: "localhost", port: 5432}, nil
	})

	// 示例2：错误提供者
	fmt.Println("\n=== 错误提供者演示 ===")

	// 演示配置错误
	fmt.Println("尝试注入 Config (期望错误):")
	err = dix.Inject(errorContainer, func(c *Config) {
		// 这个函数不会被调用，因为提供者会失败
	})
	if err != nil {
		log.Printf("预期的配置错误: %v", err)
	}

	// 演示数据库错误
	fmt.Println("尝试注入 Database (期望错误):")
	err = dix.Inject(errorContainer, func(d Database) {
		// 这个函数不会被调用，因为提供者会失败
	})
	if err != nil {
		log.Printf("预期的数据库错误: %v", err)
	}

	fmt.Println("\n=== Mixed Success/Error Demo ===")

	mixedContainer := dix.New()

	// 成功的提供者 - 返回指针类型
	dix.Provide(mixedContainer, func() (*string, error) {
		value := "success value"
		return &value, nil
	})

	// 失败的提供者 - 返回指针类型
	dix.Provide(mixedContainer, func() (*int, error) {
		return nil, errors.New("failed to get integer value")
	})

	// 示例3：混合场景
	fmt.Println("\n=== 混合提供者演示 ===")

	// 成功的提供者
	fmt.Println("获取成功的字符串:")
	var str *string
	err = dix.Inject(mixedContainer, func(s *string) {
		str = s
	})
	if err != nil {
		log.Printf("字符串注入失败: %v", err)
	} else {
		fmt.Printf("字符串注入成功: %s\n", *str)
	}

	// 失败的提供者
	fmt.Println("尝试获取失败的int:")
	err = dix.Inject(mixedContainer, func(i *int) {
		// 不会执行到这里
	})
	if err != nil {
		log.Printf("预期的int错误: %v", err)
	}

	fmt.Println("\n=== Dependency Graph ===")
	graph := dix.GetGraph(container)
	fmt.Printf("Providers:\n%s\n", graph.Providers)
	fmt.Printf("Objects:\n%s\n", graph.Objects)
}
