package main

import (
	"errors"
	"fmt"

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

	// 获取用户服务（应该成功）
	userService, err := dix.Get[*UserService](container)
	if err != nil {
		fmt.Printf("Error getting user service: %v\n", err)
		return
	}

	fmt.Printf("Successfully created user service with config: %+v\n", userService.Config)

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

	// 尝试获取配置（应该失败）
	_, err = dix.Get[*Config](errorContainer)
	if err != nil {
		fmt.Printf("Expected error when getting config: %v\n", err)
	}

	// 尝试获取数据库（应该失败，因为配置失败）
	_, err = dix.Get[Database](errorContainer)
	if err != nil {
		fmt.Printf("Expected error when getting database: %v\n", err)
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

	// 获取成功的值
	strValue, err := dix.Get[*string](mixedContainer)
	if err != nil {
		fmt.Printf("Unexpected error getting string: %v\n", err)
	} else {
		fmt.Printf("Successfully got string value: %s\n", *strValue)
	}

	// 获取失败的值
	_, err = dix.Get[*int](mixedContainer)
	if err != nil {
		fmt.Printf("Expected error getting int: %v\n", err)
	}

	fmt.Println("\n=== Dependency Graph ===")
	graph := dix.GetGraph(container)
	fmt.Printf("Providers:\n%s\n", graph.Providers)
	fmt.Printf("Objects:\n%s\n", graph.Objects)
}
