// 【功能】dixhttp 依赖图可视化:端到端综合示例。
//
// 【原理】这是一个大而全的演示(接口绑定、map/list 聚合、结构体多输出、
// 多层架构、运行时诊断),并在启动阶段刻意触发多类可识别错误
// (缺失依赖、provider 返回 error、provider panic、超时、循环依赖),
// 用于验证 dixhttp 的 /api/errors 诊断能力。
// 入门请先看 example/func 与 example/struct-in。
//
// 【运行】
//
//	cd example/http && go run .
//	# 或:task web-demo
//
// 【可选环境变量】
//
//	DIX_HTTP_ADDR=:8080                      # 服务监听地址
//	DIX_TRACE_DI=true                        # 控制台逐步 DI trace
//	DIX_DIAG_FILE=.local/dix-diag.jsonl      # JSONL 诊断文件
//
// 【预期行为】启动后打印 API 端点清单并阻塞服务,
// 浏览器打开 http://localhost:8080 查看依赖图与诊断页。
package main

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/v2/dixhttp"
)

// ==================== 接口定义 ====================

// Database 数据库接口
type Database interface {
	Connect() error
	Query(sql string) ([]map[string]any, error)
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

func (m *MySQLDatabase) Query(sql string) ([]map[string]any, error) {
	m.Logger.Info("Executing query: " + sql)
	return []map[string]any{}, nil
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

// StartupMissingDependency 用于模拟“注入缺失依赖”
type StartupMissingDependency struct{}

// StartupResolveInputMissing 用于模拟“provider 输入依赖缺失”
type StartupResolveInputMissing struct{}

// StartupResolveInputProbe 用于触发 StartupResolveInputMissing 的解析
type StartupResolveInputProbe struct{}

// StartupBrokenComponent 用于模拟“provider 返回 error”
type StartupBrokenComponent struct{}

// StartupPanicComponent 用于模拟“provider panic”
type StartupPanicComponent struct{}

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

func isMachineDiagMode() bool {
	mode := strings.TrimSpace(strings.ToLower(os.Getenv("DIX_LLM_DIAG_MODE")))
	switch mode {
	case "only", "machine", "machine-only", "machine_only", "json":
		return true
	default:
		return false
	}
}

func configureExampleLogOutput() {
	if isMachineDiagMode() {
		log.SetOutput(io.Discard)
	}
}

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
	log.Println("   - GET /api/runtime-stats - Provider startup runtime stats")
	log.Println("   - GET /api/errors - Recent Inject/TryInject errors")
	log.Println("   - GET /api/trace - In-memory dixtrace timeline query")
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

func logStartupScenarioResult(di *dix.Dix, scenario string, err error, previousCount int) {
	if err == nil {
		return
	}

	recent := di.GetRecentErrors(0)
	if len(recent) == 0 {
		log.Printf("⚠️ [startup-diagnostic][%s] err=%v (no recent error records)", scenario, err)
		return
	}

	newCount := len(recent) - previousCount
	if newCount <= 0 {
		newCount = 1
	}
	if newCount > len(recent) {
		newCount = len(recent)
	}

	log.Printf("⚠️ [startup-diagnostic][%s] captured %d record(s)", scenario, newCount)
	for i := newCount - 1; i >= 0; i-- {
		item := recent[i]
		recordIndex := newCount - i
		log.Printf("   - record=%d", recordIndex)
		log.Printf("     type=%s", item.ErrorType)
		log.Printf("     op=%s", item.Operation)
		if item.Stage != "" {
			log.Printf("     stage=%s", item.Stage)
		}
		log.Printf("     message=%s", item.Message)
		if item.ProviderFunction != "" {
			log.Printf("     provider=%s", item.ProviderFunction)
		}
		if item.OutputType != "" {
			log.Printf("     output=%s", item.OutputType)
		}
		if item.InputType != "" {
			log.Printf("     input=%s", item.InputType)
		}
		if item.Hint != "" {
			log.Printf("     hint=%s", item.Hint)
		}
	}
}

func runStartupErrorScenarios(di *dix.Dix) {
	log.Println("🧪 Running startup error diagnostics...")

	before := len(di.GetRecentErrors(0))
	if err := di.TryProvide(nil); err != nil {
		logStartupScenarioResult(di, "invalid_provider_registration", err, before)
	}

	before = len(di.GetRecentErrors(0))
	if err := di.TryInject(func(*StartupMissingDependency) {}); err != nil {
		logStartupScenarioResult(di, "inject_missing_dependency", err, before)
	}

	dix.Provide(di, func(*StartupResolveInputMissing) *StartupResolveInputProbe {
		return &StartupResolveInputProbe{}
	})
	before = len(di.GetRecentErrors(0))
	if err := di.TryInject(func(*StartupResolveInputProbe) {}); err != nil {
		logStartupScenarioResult(di, "provider_input_unresolved", err, before)
	}

	dix.Provide(di, func(logger Logger) (*StartupBrokenComponent, error) {
		logger.Info("[demo] StartupBrokenComponent returns intentional error")
		return nil, errors.New("demo startup: provider return error")
	})
	before = len(di.GetRecentErrors(0))
	if err := di.TryInject(func(*StartupBrokenComponent) {}); err != nil {
		logStartupScenarioResult(di, "provider_return_error", err, before)
	}

	before = len(di.GetRecentErrors(0))
	if err := di.TryInject(func(logger Logger) error {
		logger.Info("[demo] inject callback returns intentional error")
		return errors.New("demo startup: inject callback error")
	}); err != nil {
		logStartupScenarioResult(di, "inject_callback_error", err, before)
	}

	dix.Provide(di, func(logger Logger) *StartupPanicComponent {
		logger.Info("[demo] StartupPanicComponent panics intentionally")
		panic("demo startup: provider panic")
	})
	before = len(di.GetRecentErrors(0))
	if err := di.TryInject(func(*StartupPanicComponent) {}); err != nil {
		logStartupScenarioResult(di, "provider_panic", err, before)
	}

	before = len(di.GetRecentErrors(0))
	if err := di.TryInject(func(*TimeoutProbe) {}); err != nil {
		logStartupScenarioResult(di, "provider_timeout", err, before)
	}

	// 循环依赖演示使用临时容器，避免污染主容器导致后续所有注入失败。
	cycleDI := dix.New()
	type cycleA struct{}
	type cycleB struct{}
	type cycleC struct{}
	dix.Provide(cycleDI, func(*cycleC) *cycleA { return &cycleA{} })
	dix.Provide(cycleDI, func(*cycleA) *cycleB { return &cycleB{} })
	dix.Provide(cycleDI, func(*cycleB) *cycleC { return &cycleC{} })
	beforeCycle := len(cycleDI.GetRecentErrors(0))
	if err := cycleDI.TryInject(func(*cycleA) {}); err != nil {
		logStartupScenarioResult(cycleDI, "dependency_cycle(temp_container)", err, beforeCycle)
	}

	log.Println("🧪 Startup diagnostics done. Visit /api/errors to verify error_type/hint recognition.")
}

func main() {
	configureExampleLogOutput()

	di := buildContainer()
	preCreateObjects(di)

	// 在启动阶段统一触发多类可识别错误，便于验证 error_type/hint 的识别与定位
	runStartupErrorScenarios(di)

	log.Println("")

	// ==================== 启动HTTP服务器 ====================

	// Create HTTP server for visualization
	server := dixhttp.NewServer(di)

	if err := startVisualizationServer(server); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server error:", err)
	}
}

// buildContainer 注册示例的全部组件并返回容器:
// 基础组件 → 服务层 → 控制器层 → Application 聚合,
// 另含用于诊断演示的慢依赖/错误场景 provider。
// 该装配契约由 main_test.go 的 TestBuildContainerWiresApplication 锁定。
func buildContainer() *dix.Dix {
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

	return di
}

// preCreateObjects 通过函数注入触发 provider 执行,
// 让 dixhttp 的 objects 视图在启动后即有内容可展示。
func preCreateObjects(di *dix.Dix) {
	log.Println("📦 Pre-creating objects for visualization...")

	if err := di.TryInject(func(
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
	}); err != nil {
		log.Printf("⚠️ pre-create injection failed, web will still start for diagnostics: %v", err)
	}
}
