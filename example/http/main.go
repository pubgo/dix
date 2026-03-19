package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/v2/dixhttp"
	"github.com/pubgo/dix/v2/dixinternal"
)

// ==================== 接口定义 ====================

// Database 数据库接口
type Database interface {
	Connect() error
	Query(sql string) ([]map[string]interface{}, error)
}

// Cache 缓存接口
type Cache interface {
	Get(key string) (string, error)
	Set(key, value string, ttl time.Duration) error
}

// Logger 日志接口
type Logger interface {
	Info(msg string)
	Error(msg string)
}

// HTTPClient HTTP客户端接口
type HTTPClient interface {
	Get(url string) ([]byte, error)
	Post(url string, data []byte) ([]byte, error)
}

// Service 服务接口（所有业务服务都实现此接口）
type Service interface {
	Name() string
}

// ==================== 实现 ====================

// MySQLDatabase MySQL数据库实现
type MySQLDatabase struct {
	Host     string
	Port     int
	Database string
	Logger   Logger
}

func (m *MySQLDatabase) Connect() error {
	m.Logger.Info("Connecting to MySQL database")
	return nil
}

func (m *MySQLDatabase) Query(sql string) ([]map[string]interface{}, error) {
	m.Logger.Info("Executing query: " + sql)
	return []map[string]interface{}{}, nil
}

// RedisCache Redis缓存实现
type RedisCache struct {
	Host   string
	Port   int
	Logger Logger
}

func (r *RedisCache) Get(key string) (string, error) {
	r.Logger.Info("Getting from cache: " + key)
	return "", nil
}

func (r *RedisCache) Set(key, value string, ttl time.Duration) error {
	r.Logger.Info("Setting cache: " + key)
	return nil
}

// FileCache 文件缓存实现
type FileCache struct {
	Path   string
	Logger Logger
}

func (f *FileCache) Get(key string) (string, error) {
	f.Logger.Info("Getting from file cache: " + key)
	return "", nil
}

func (f *FileCache) Set(key, value string, ttl time.Duration) error {
	f.Logger.Info("Setting file cache: " + key)
	return nil
}

// ConsoleLogger 控制台日志实现
type ConsoleLogger struct {
	Level string
}

func (c *ConsoleLogger) Info(msg string) {
	log.Printf("[INFO] %s", msg)
}

func (c *ConsoleLogger) Error(msg string) {
	log.Printf("[ERROR] %s", msg)
}

// DefaultHTTPClient 默认HTTP客户端实现
type DefaultHTTPClient struct {
	Timeout time.Duration
	Logger  Logger
}

func (d *DefaultHTTPClient) Get(url string) ([]byte, error) {
	d.Logger.Info("HTTP GET: " + url)
	return []byte("response"), nil
}

func (d *DefaultHTTPClient) Post(url string, data []byte) ([]byte, error) {
	d.Logger.Info("HTTP POST: " + url)
	return []byte("response"), nil
}

// ==================== 配置结构体 ====================

// AppConfig 应用配置
type AppConfig struct {
	Database *DatabaseConfig
	Cache    *CacheConfig
	HTTP     *HTTPConfig
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string
	Port     int
	Database string
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Type string // "redis" or "file"
	Host string
	Port int
	Path string
}

// HTTPConfig HTTP配置
type HTTPConfig struct {
	Timeout time.Duration
}

// ==================== 服务层 ====================

// UserService 用户服务
type UserService struct {
	DB     Database
	Cache  Cache
	Logger Logger
}

func (u *UserService) Name() string {
	return "UserService"
}

// OrderService 订单服务
type OrderService struct {
	DB          Database
	Cache       Cache
	UserService *UserService
	Logger      Logger
}

func (o *OrderService) Name() string {
	return "OrderService"
}

// PaymentService 支付服务
type PaymentService struct {
	HTTPClient HTTPClient
	Logger     Logger
}

func (p *PaymentService) Name() string {
	return "PaymentService"
}

// NotificationService 通知服务
type NotificationService struct {
	HTTPClient HTTPClient
	Cache      Cache
	Logger     Logger
}

func (n *NotificationService) Name() string {
	return "NotificationService"
}

// SlowRemoteClient 模拟一个外部慢依赖
type SlowRemoteClient struct {
	Ready bool
}

// TimeoutProbe 仅用于触发 SlowRemoteClient 的构建
type TimeoutProbe struct {
	Client *SlowRemoteClient
}

// ==================== 业务逻辑层 ====================

// UserController 用户控制器
type UserController struct {
	UserService *UserService
	Logger      Logger
}

// OrderController 订单控制器
type OrderController struct {
	OrderService        *OrderService
	PaymentService      *PaymentService
	NotificationService *NotificationService
	Logger              Logger
}

// ==================== 应用主结构 ====================

// Application 应用主结构
type Application struct {
	Config              *AppConfig
	UserController      *UserController
	OrderController     *OrderController
	NotificationService *NotificationService
	AllServices         []Service          // 所有服务的集合（使用具体接口类型）
	ServiceMap          map[string]Service // 服务映射（使用具体接口类型）
}

const defaultHTTPAddr = ":8080"

func startVisualizationServer(server *dixhttp.Server) error {
	addr := os.Getenv("DIX_HTTP_ADDR")
	if addr == "" {
		addr = defaultHTTPAddr
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		if addr == defaultHTTPAddr {
			log.Printf("⚠️ Port %s unavailable (%v), trying a random available port...", addr, err)
			ln, err = net.Listen("tcp", ":0")
		}
		if err != nil {
			return err
		}
	}

	actualAddr := ln.Addr().String()
	displayAddr := actualAddr
	if _, port, splitErr := net.SplitHostPort(actualAddr); splitErr == nil && port != "" {
		displayAddr = "localhost:" + port
	}

	log.Printf("🚀 Starting HTTP server on http://%s", displayAddr)
	log.Printf("📊 Open http://%s in your browser to view dependency relationships", displayAddr)
	log.Println("📡 API endpoints:")
	log.Println("   - GET /api/dependencies - JSON data of dependencies")
	log.Println("   - GET /api/graph?type=providers - DOT graph format")
	log.Println("   - GET /api/graph?type=provider_types - Provider types graph")
	log.Println("   - GET /api/graph?type=objects - Objects graph")
	log.Println("")
	log.Println("💡 This example demonstrates:")
	log.Println("   - Interface-based dependency injection")
	log.Println("   - Multiple implementations (RedisCache, FileCache)")
	log.Println("   - Map and Slice dependencies")
	log.Println("   - Struct output (auto-flattening)")
	log.Println("   - Multi-layer architecture (Config -> Services -> Controllers -> Application)")
	log.Println("   - Complex dependency chains")
	log.Println("   - Simulated timeout provider (for red-highlight diagnosis)")
	log.Println("")
	log.Println("💡 Runtime stats will include a timeout provider sample for diagnostics!")

	httpServer := &http.Server{Handler: server}
	return httpServer.Serve(ln)
}

func main() {
	// Create a Dix container
	di := dix.New(
		dix.WithProviderTimeout(200*time.Millisecond),
		dix.WithSlowProviderThreshold(80*time.Millisecond),
	)

	// ==================== 注册基础组件 ====================

	// 注册日志服务
	dix.Provide(di, func() Logger {
		return &ConsoleLogger{Level: "info"}
	})

	// 注册配置（结构体输出会自动分解为各个字段类型）
	dix.Provide(di, func() AppConfig {
		return AppConfig{
			Database: &DatabaseConfig{
				Host:     "localhost",
				Port:     3306,
				Database: "mydb",
			},
			Cache: &CacheConfig{
				Type: "redis",
				Host: "localhost",
				Port: 6379,
				Path: "/tmp/cache",
			},
			HTTP: &HTTPConfig{
				Timeout: 30 * time.Second,
			},
		}
	})

	// 注册 AppConfig 指针类型（Application 需要 *AppConfig）
	dix.Provide(di, func(config AppConfig) *AppConfig {
		return &config
	})

	// 注册数据库实现
	dix.Provide(di, func(logger Logger, config *DatabaseConfig) Database {
		return &MySQLDatabase{
			Host:     config.Host,
			Port:     config.Port,
			Database: config.Database,
			Logger:   logger,
		}
	})

	// 注册多个缓存实现（使用 map）
	dix.Provide(di, func(logger Logger, config *CacheConfig) map[string]Cache {
		return map[string]Cache{
			"redis": &RedisCache{
				Host:   config.Host,
				Port:   config.Port,
				Logger: logger,
			},
			"file": &FileCache{
				Path:   config.Path,
				Logger: logger,
			},
		}
	})

	// 注册HTTP客户端
	dix.Provide(di, func(logger Logger, config *HTTPConfig) HTTPClient {
		return &DefaultHTTPClient{
			Timeout: config.Timeout,
			Logger:  logger,
		}
	})

	// ==================== 模拟异常场景（用于可视化排查） ====================

	// 模拟慢依赖：刻意 sleep 超过 ProviderTimeout，触发 timeout
	dix.Provide(di, func(logger Logger) *SlowRemoteClient {
		logger.Info("[demo] SlowRemoteClient start (expected timeout)")
		time.Sleep(450 * time.Millisecond)
		logger.Info("[demo] SlowRemoteClient done")
		return &SlowRemoteClient{Ready: true}
	})

	// 触发 SlowRemoteClient 的解析
	dix.Provide(di, func(client *SlowRemoteClient) *TimeoutProbe {
		return &TimeoutProbe{Client: client}
	})

	// ==================== 注册服务层 ====================

	// 注册用户服务
	dix.Provide(di, func(db Database, cache map[string]Cache, logger Logger) *UserService {
		return &UserService{
			DB:     db,
			Cache:  cache["redis"], // 使用redis缓存
			Logger: logger,
		}
	})

	// 注册订单服务（依赖用户服务）
	dix.Provide(di, func(db Database, cache map[string]Cache, userService *UserService, logger Logger) *OrderService {
		return &OrderService{
			DB:          db,
			Cache:       cache["redis"],
			UserService: userService,
			Logger:      logger,
		}
	})

	// 注册支付服务
	dix.Provide(di, func(httpClient HTTPClient, logger Logger) *PaymentService {
		return &PaymentService{
			HTTPClient: httpClient,
			Logger:     logger,
		}
	})

	// 注册通知服务（依赖HTTP客户端和缓存）
	dix.Provide(di, func(httpClient HTTPClient, cache map[string]Cache, logger Logger) *NotificationService {
		return &NotificationService{
			HTTPClient: httpClient,
			Cache:      cache["file"], // 使用文件缓存
			Logger:     logger,
		}
	})

	// ==================== 注册控制器层 ====================

	// 注册用户控制器
	dix.Provide(di, func(userService *UserService, logger Logger) *UserController {
		return &UserController{
			UserService: userService,
			Logger:      logger,
		}
	})

	// 注册订单控制器（依赖多个服务）
	dix.Provide(di, func(
		orderService *OrderService,
		paymentService *PaymentService,
		notificationService *NotificationService,
		logger Logger,
	) *OrderController {
		return &OrderController{
			OrderService:        orderService,
			PaymentService:      paymentService,
			NotificationService: notificationService,
			Logger:              logger,
		}
	})

	// ==================== 注册应用主结构 ====================

	// 注册所有服务的列表（使用 slice，使用具体接口类型）
	dix.Provide(di, func(
		userService *UserService,
		orderService *OrderService,
		paymentService *PaymentService,
		notificationService *NotificationService,
	) []Service {
		return []Service{
			userService,
			orderService,
			paymentService,
			notificationService,
		}
	})

	// 注册服务映射（使用 map，使用具体接口类型）
	dix.Provide(di, func(
		userService *UserService,
		orderService *OrderService,
		paymentService *PaymentService,
		notificationService *NotificationService,
	) map[string]Service {
		return map[string]Service{
			"user":         userService,
			"order":        orderService,
			"payment":      paymentService,
			"notification": notificationService,
		}
	})

	// 注册应用主结构（包含所有依赖）
	dix.Provide(di, func(
		config *AppConfig,
		userController *UserController,
		orderController *OrderController,
		notificationService *NotificationService,
		allServices []Service,
		serviceMap map[string]Service,
	) *Application {
		return &Application{
			Config:              config,
			UserController:      userController,
			OrderController:     orderController,
			NotificationService: notificationService,
			AllServices:         allServices,
			ServiceMap:          serviceMap,
		}
	})

	// ==================== 预创建对象（用于可视化） ====================
	// 注意：Objects 只有在实际执行 Provider 后才会被创建
	// 这里我们通过函数注入来触发对象的创建，这样 objects 视图才能显示内容

	log.Println("📦 Pre-creating objects for visualization...")

	// 使用函数注入来创建对象（这种方式可以处理指针类型）
	dix.Inject(di, func(
		app *Application,
		userService *UserService,
		orderService *OrderService,
		db Database,
		cacheMap map[string]Cache,
		allServices []Service,
	) {
		if app != nil {
			log.Printf("✅ Application created successfully")
			log.Printf("   - UserController: %v", app.UserController != nil)
			log.Printf("   - OrderController: %v", app.OrderController != nil)
			log.Printf("   - Services count: %d", len(app.AllServices))
			log.Printf("   - ServiceMap keys: %d", len(app.ServiceMap))
		}

		if userService != nil {
			log.Printf("✅ UserService created")
		}

		if orderService != nil {
			log.Printf("✅ OrderService created")
		}

		if db != nil {
			log.Printf("✅ Database interface created")
		}

		if cacheMap != nil {
			log.Printf("✅ Cache map created with %d entries", len(cacheMap))
		}

		if allServices != nil {
			log.Printf("✅ Services list created with %d services", len(allServices))
		}
	})

	// 模拟一次超时 provider 执行，用于在可视化中展示“超时标红”
	if err := di.TryInject(func(*TimeoutProbe) {}); err != nil {
		log.Printf("⚠️ [demo] expected timeout case captured: %v", err)
	}

	log.Println("")

	// ==================== 启动HTTP服务器 ====================

	// Create HTTP server for visualization
	server := dixhttp.NewServer((*dixinternal.Dix)(di))

	if err := startVisualizationServer(server); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server error:", err)
	}
}
