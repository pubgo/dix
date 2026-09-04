// 【功能】dixhttp 依赖图可视化:大规模端到端综合示例。
//
// 【原理】构造真实项目规模的容器(十个域模块 + 泛型插件/工作器族,
// 约 200 个 provider、300 个对象),并触发多类可诊断错误,用于:
//   - 体验五视图 UI(概览/依赖图/检索/调用链/诊断)的交互流程;
//   - 验证大规模下"模块下钻 + 邻域子图 + 服务端检索"的可用性。
//
// 入门请先看 example/inject-func 与 example/inject-struct。
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
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/v2/dixhttp"

	"github.com/pubgo/dix/example/http/domain/analytics"
	"github.com/pubgo/dix/example/http/domain/billing"
	"github.com/pubgo/dix/example/http/domain/identity"
	"github.com/pubgo/dix/example/http/domain/inventory"
	"github.com/pubgo/dix/example/http/domain/media"
	"github.com/pubgo/dix/example/http/domain/notification"
	"github.com/pubgo/dix/example/http/domain/searchx"
	"github.com/pubgo/dix/example/http/domain/shipping"
	"github.com/pubgo/dix/example/http/domain/storage"
	"github.com/pubgo/dix/example/http/domain/workflow"
)

// Logger 全局日志接口。
type Logger interface {
	Info(msg string)
	Error(msg string)
}

// ConsoleLogger 默认实现。
type ConsoleLogger struct{ Prefix string }

func (c *ConsoleLogger) Info(msg string)  { log.Printf("[INFO] %s", msg) }
func (c *ConsoleLogger) Error(msg string) { log.Printf("[ERROR] %s", msg) }

const defaultHTTPAddr = ":8080"

// ==================== 诊断演示用的合成组件 ====================

// SlowRemoteClient 模拟外部慢依赖(触发 provider_timeout)。
type SlowRemoteClient struct{ Ready bool }

// TimeoutProbe 触发 SlowRemoteClient 的解析。
type TimeoutProbe struct{ Client *SlowRemoteClient }

// StartupMissingDependency 模拟注入缺失依赖。
type StartupMissingDependency struct{}

// StartupResolveInputMissing 模拟 provider 输入依赖缺失。
type StartupResolveInputMissing struct{}

// StartupResolveInputProbe 用于触发 StartupResolveInputMissing 的解析。
type StartupResolveInputProbe struct{}

// StartupBrokenComponent 模拟 provider 返回 error。
type StartupBrokenComponent struct{}

// StartupPanicComponent 模拟 provider panic。
type StartupPanicComponent struct{}

// ==================== 应用主结构 ====================

// Application 应用聚合根:引用十个域模块的服务,是整棵依赖图的汇点。
type Application struct {
	Logger    Logger
	Billing   *billing.Service
	Inventory *inventory.Service
	Shipping  *shipping.Service
	Identity  *identity.Service
	Analytics *analytics.Service
	Notify    *notification.Service
	Search    *searchx.Service
	Storage   *storage.Service
	Media     *media.Service
	Workflow  *workflow.Service
	Plugins   []string
}

// ==================== HTTP 服务器 ====================

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
	log.Printf("📊 Open http://%s in your browser: overview / graph / search / trace / diag", displayAddr)
	log.Println("📡 API endpoints:")
	log.Println("   - GET /api/dependencies - JSON data of dependencies")
	log.Println("   - GET /api/modules      - module-level aggregation")
	log.Println("   - GET /api/ego          - neighborhood subgraph")
	log.Println("   - GET /api/search       - server-side graph search")
	log.Println("   - GET /api/stats        - overview statistics")
	log.Println("   - GET /api/runtime-stats - provider startup timings")
	log.Println("   - GET /api/errors       - recent inject errors")
	log.Println("   - GET /api/diagnostics  - DIX_DIAG_FILE records")
	log.Println("   - GET /api/trace        - dixtrace event query")
	log.Println("   - GET /api/trace-tree   - nested call tree per trace")
	return (&http.Server{Handler: server}).Serve(ln)
}

// ==================== 启动诊断场景 ====================

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
		log.Printf("   - record=%d type=%s op=%s message=%s",
			newCount-i, item.ErrorType, item.Operation, item.Message)
		if item.Hint != "" {
			log.Printf("     hint=%s", item.Hint)
		}
	}
}

// runStartupErrorScenarios 在启动阶段统一触发多类可识别错误,
// 验证 /api/errors 的 error_type/hint 识别与定位能力。
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

	// 循环依赖演示使用临时容器,避免污染主容器。
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

// buildContainer 注册示例的全部组件并返回容器:
// 十个域模块(billing/workflow/…) + 泛型插件/工作器族 + 聚合根 Application,
// 另含用于诊断演示的慢依赖/错误场景 provider。
// 装配契约由 main_test.go 的 TestBuildContainerWiresApplication 锁定。
func buildContainer() *dix.Dix {
	di := dix.New(
		dix.WithProviderTimeout(200*time.Millisecond),
		dix.WithSlowProviderThreshold(80*time.Millisecond),
	)

	// 基础组件
	dix.Provide(di, func() Logger { return &ConsoleLogger{Prefix: "app"} })

	// 十个域模块:每域五层链路(配置→客户端→仓储→服务→处理器)+ 多区域连接
	analytics.Providers(di)
	billing.Providers(di)
	identity.Providers(di)
	inventory.Providers(di)
	media.Providers(di)
	notification.Providers(di)
	searchx.Providers(di)
	shipping.Providers(di)
	storage.Providers(di)
	workflow.Providers(di)

	// 泛型插件/工作器族(百余合成依赖,形成 Plugin -> Worker 链路)
	activators := registerPlugins(di)

	// 聚合根:引用全部域服务,是依赖图的汇点
	dix.Provide(di, func(
		logger Logger,
		billing *billing.Service,
		inventory *inventory.Service,
		shipping *shipping.Service,
		identity *identity.Service,
		analytics *analytics.Service,
		notify *notification.Service,
		search *searchx.Service,
		storage *storage.Service,
		media *media.Service,
		workflow *workflow.Service,
	) *Application {
		return &Application{
			Logger:    logger,
			Billing:   billing,
			Inventory: inventory,
			Shipping:  shipping,
			Identity:  identity,
			Analytics: analytics,
			Notify:    notify,
			Search:    search,
			Storage:   storage,
			Media:     media,
			Workflow:  workflow,
			Plugins:   pluginNames,
		}
	})

	// 预创建对象:执行插件 provider,让 objects 视图有内容
	for _, activate := range activators {
		activate()
	}

	// 模拟慢依赖:刻意超过 ProviderTimeout(用于可视化排查)
	dix.Provide(di, func(logger Logger) *SlowRemoteClient {
		logger.Info("[demo] SlowRemoteClient start (expected timeout)")
		time.Sleep(450 * time.Millisecond)
		return &SlowRemoteClient{Ready: true}
	})
	dix.Provide(di, func(client *SlowRemoteClient) *TimeoutProbe {
		return &TimeoutProbe{Client: client}
	})

	// 错误场景 provider(诊断演示)
	dix.Provide(di, func(logger Logger) (*StartupBrokenComponent, error) {
		return nil, errors.New("demo startup: provider return error")
	})
	dix.Provide(di, func(logger Logger) *StartupPanicComponent {
		panic("demo startup: provider panic")
	})

	return di
}

// preCreateObjects 通过函数注入触发核心对象创建,
// 让 dixhttp 的 objects 视图在启动后即有内容可展示。
func preCreateObjects(di *dix.Dix) {
	if err := di.TryInject(func(app *Application, logger Logger) {
		log.Printf("✅ Application created: modules=10 plugins=%d", len(app.Plugins))
	}); err != nil {
		log.Printf("⚠️ pre-create injection failed: %v", err)
	}
}

func main() {
	di := buildContainer()
	preCreateObjects(di)

	// 启动阶段触发多类可识别错误,验证 error_type/hint 识别
	runStartupErrorScenarios(di)

	log.Println("")

	server := dixhttp.NewServer(di)
	if err := startVisualizationServer(server); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server error:", err)
	}
}
