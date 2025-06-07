package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pubgo/dix"
	"github.com/pubgo/funk/errors"
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
			return nil, errors.Wrap(err, "failed to connect to database")
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

	// 示例1：正常获取实例
	fmt.Println("=== 正常实例获取 ===")
	var userService *UserService
	_, err := dix.Inject(container, func(us *UserService) {
		userService = us
	})
	if err != nil {
		log.Printf("注入失败: %v", err)
		return
	}
	fmt.Printf("UserService 实例获取成功，配置: %+v\n", userService.Config)

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

	// 示例2：错误提供者
	fmt.Println("\n=== 错误提供者演示 ===")

	// 演示配置错误
	fmt.Println("尝试注入 Config (期望错误):")
	_, err = dix.Inject(errorContainer, func(c *Config) {
		// 这个函数不会被调用，因为提供者会失败
	})
	if err != nil {
		log.Printf("预期的配置错误: %v", err)
	}

	// 演示数据库错误
	fmt.Println("尝试注入 Logger (期望错误):")
	_, err = dix.Inject(errorContainer, func(l Logger) {
		// 这个函数不会被调用，因为提供者会失败
	})
	if err != nil {
		log.Printf("预期的Logger错误: %v", err)
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

	// 示例3：混合场景
	fmt.Println("\n=== 混合提供者演示 ===")

	// 成功的提供者
	fmt.Println("获取成功的Logger:")
	var logger Logger
	_, err = dix.Inject(mixedContainer, func(l Logger) {
		logger = l
	})
	if err != nil {
		log.Printf("Logger 注入失败: %v", err)
	} else {
		logger.Log("Logger 注入成功!")
	}

	// 失败的提供者
	fmt.Println("尝试获取失败的Config:")
	_, err = dix.Inject(mixedContainer, func(c *Config) {
		// 不会执行到这里
	})
	if err != nil {
		log.Printf("预期的Config错误: %v", err)
	}

	fmt.Println("\n=== Dependency Graph ===")
	graph := dix.GetGraph(container)
	fmt.Printf("Providers:\n%s\n", graph.Providers)
	fmt.Printf("Objects:\n%s\n", graph.Objects)
}
