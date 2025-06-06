package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/pubgo/dix"
	"github.com/pubgo/funk/recovery"
)

// Logger 接口
type Logger interface {
	Log(msg string)
}

// ConsoleLogger 实现
type ConsoleLogger struct {
	prefix string
}

func (c *ConsoleLogger) Log(msg string) {
	fmt.Printf("[%s] %s\n", c.prefix, msg)
}

// Config 配置结构体
type Config struct {
	DatabaseURL string
	LogLevel    string
	Debug       bool
}

// Database 接口
type Database interface {
	Connect() error
	Query(sql string) ([]string, error)
}

// PostgresDB 实现
type PostgresDB struct {
	url string
}

func (db *PostgresDB) Connect() error {
	fmt.Printf("Connecting to database: %s\n", db.url)
	return nil
}

func (db *PostgresDB) Query(sql string) ([]string, error) {
	fmt.Printf("Executing query: %s\n", sql)
	return []string{"result1", "result2"}, nil
}

// UserService 用户服务
type UserService struct {
	Logger Logger
	DB     Database
	Config *Config
}

func main() {
	defer recovery.Exit()

	fmt.Println("=== Provider Error Handling Demo (Correct Types) ===")

	container := dix.New()

	fmt.Println("\n=== Successful Provider Chain ===")

	// 注册配置提供者（成功案例）
	dix.Provide(container, func() (*Config, error) {
		fmt.Println("Loading configuration...")

		// 模拟从环境变量或配置文件加载
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			dbURL = "postgres://localhost:5432/myapp"
		}

		config := &Config{
			DatabaseURL: dbURL,
			LogLevel:    "INFO",
			Debug:       true,
		}

		fmt.Printf("Configuration loaded: %+v\n", config)
		return config, nil // 成功返回
	})

	// 注册日志提供者
	dix.Provide(container, func(config *Config) (Logger, error) {
		fmt.Println("Creating logger...")

		if config.LogLevel == "" {
			return nil, errors.New("log level cannot be empty")
		}

		logger := &ConsoleLogger{
			prefix: config.LogLevel,
		}

		return logger, nil
	})

	// 注册数据库提供者（依赖配置）
	dix.Provide(container, func(config *Config) (Database, error) {
		fmt.Println("Creating database connection...")

		if config.DatabaseURL == "" {
			return nil, errors.New("database URL cannot be empty")
		}

		db := &PostgresDB{url: config.DatabaseURL}

		// 模拟连接测试
		if err := db.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}

		return db, nil
	})

	// 注册用户服务提供者
	dix.Provide(container, func(logger Logger, db Database, config *Config) (*UserService, error) {
		fmt.Println("Creating user service...")

		service := &UserService{
			Logger: logger,
			DB:     db,
			Config: config,
		}

		// 初始化检查
		service.Logger.Log("UserService initialized successfully")
		return service, nil
	})

	// 获取用户服务（应该成功）
	userService, err := dix.Get[*UserService](container)
	if err != nil {
		fmt.Printf("Error getting user service: %v\n", err)
		return
	}

	fmt.Printf("Successfully created user service\n")
	userService.Logger.Log("Testing database query...")

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
		fmt.Println("Attempting to load configuration...")
		return nil, errors.New("configuration service is unavailable")
	})

	// 注册依赖配置的日志提供者
	dix.Provide(errorContainer, func(config *Config) (Logger, error) {
		// 这个不会被调用，因为配置提供者会失败
		fmt.Println("This should not be printed")
		return &ConsoleLogger{prefix: "ERROR"}, nil
	})

	// 尝试获取配置（应该失败）
	_, err = dix.Get[*Config](errorContainer)
	if err != nil {
		fmt.Printf("Expected error when getting config: %v\n", err)
	}

	// 尝试获取日志（应该失败，因为配置失败）
	_, err = dix.Get[Logger](errorContainer)
	if err != nil {
		fmt.Printf("Expected error when getting logger: %v\n", err)
	}

	fmt.Println("\n=== Mixed Success/Error Demo ===")

	mixedContainer := dix.New()

	// 成功的提供者 - 返回接口
	dix.Provide(mixedContainer, func() (Logger, error) {
		return &ConsoleLogger{prefix: "SUCCESS"}, nil
	})

	// 失败的提供者 - 返回指针
	dix.Provide(mixedContainer, func() (*Config, error) {
		return nil, errors.New("failed to load configuration")
	})

	// 获取成功的值
	logger, err := dix.Get[Logger](mixedContainer)
	if err != nil {
		fmt.Printf("Unexpected error getting logger: %v\n", err)
	} else {
		logger.Log("Successfully got logger from mixed container")
	}

	// 获取失败的值
	_, err = dix.Get[*Config](mixedContainer)
	if err != nil {
		fmt.Printf("Expected error getting config: %v\n", err)
	}

	fmt.Println("\n=== Dependency Graph ===")
	graph := dix.GetGraph(container)
	fmt.Printf("Providers:\n%s\n", graph.Providers)
	fmt.Printf("Objects:\n%s\n", graph.Objects)
}
