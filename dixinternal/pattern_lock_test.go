package dixinternal

// 模式锁测试(behavior lock tests)。
//
// 与 example/ 目录的示例一一对应,把 provider/inject 的公开行为契约用断言锁死,
// 防止示例演示的语义在后续重构中被无声破坏。语义依据:
//   - getOutputTypeValues:某类型的全部 provider 按注册顺序执行,产物缓存
//   - getValue:单值取同类型产物列表的最后一个;map/list 为聚合查询
//   - handleOutput/makeMap/makeList:struct 输出按导出字段拆分注册,
//     map 输出按 key 分组(同 key 取最后),nil 产物跳过
//   - inject:DixInject 前缀方法在字段注入之前按方法名字母序执行

import (
	"errors"
	"strings"
	"testing"
)

func mustProvideLock(t *testing.T, di *Dix, fn any) {
	t.Helper()
	if err := di.TryProvide(fn); err != nil {
		t.Fatalf("TryProvide(%T): %v", fn, err)
	}
}

// 锁定 example/lazy:
//  1. provider 惰性执行:仅在 Inject 需要其产物时才运行;
//  2. 同一输出类型的多个 provider 全部执行(注册顺序),单值注入取最后注册者的产物;
//  3. 产物缓存在容器内,后续 Inject 不再重复执行 provider。
func TestPatternLazyResolution(t *testing.T) {
	di := New()

	type Service struct{ Name string }

	var execOrder []string
	var svcA, svcB, gotByC *Service
	errSentinel := errors.New("ready")

	mustProvideLock(t, di, func() *Service {
		execOrder = append(execOrder, "A")
		svcA = &Service{Name: "A"}
		return svcA
	})
	mustProvideLock(t, di, func() *Service {
		execOrder = append(execOrder, "B")
		svcB = &Service{Name: "B"}
		return svcB
	})
	mustProvideLock(t, di, func(s *Service) error {
		execOrder = append(execOrder, "C")
		gotByC = s
		return errSentinel
	})

	if len(execOrder) != 0 {
		t.Fatalf("providers must not run before the first Inject, got %v", execOrder)
	}

	var injected error
	if err := di.TryInject(func(err error) { injected = err }); err != nil {
		t.Fatalf("TryInject: %v", err)
	}

	if !errors.Is(injected, errSentinel) {
		t.Fatalf("injected error = %v, want sentinel %v", injected, errSentinel)
	}
	if got, want := strings.Join(execOrder, ","), "A,B,C"; got != want {
		t.Fatalf("execution order = %q, want %q", got, want)
	}
	if gotByC != svcB {
		t.Fatal("single-value resolution must use the last registered provider's value")
	}

	before := len(execOrder)
	if err := di.TryInject(func(err error) { injected = err }); err != nil {
		t.Fatalf("second TryInject: %v", err)
	}
	if len(execOrder) != before {
		t.Fatalf("providers re-executed on second inject: %v", execOrder)
	}
}

// 锁定 example/inject-func:同类型多 provider 时,单值注入取最后注册者,
// 列表注入按注册顺序聚合全部产物;两类查询互不干扰。
func TestPatternSingleValueLastProviderWins(t *testing.T) {
	di := New()

	type Greet func() string

	var calls int
	mustProvideLock(t, di, func() Greet {
		calls++
		return func() string { return "hello" }
	})
	mustProvideLock(t, di, func() Greet {
		calls++
		return func() string { return "world" }
	})

	var single Greet
	var all []Greet
	if err := di.TryInject(func(g Greet, gs []Greet) { single, all = g, gs }); err != nil {
		t.Fatalf("TryInject: %v", err)
	}

	if calls != 2 {
		t.Fatalf("provider executions = %d, want 2", calls)
	}
	if single() != "world" {
		t.Fatalf("single value = %q, want latest registered %q", single(), "world")
	}
	if len(all) != 2 || all[0]() != "hello" || all[1]() != "world" {
		t.Fatalf("aggregate = [%s %s], want registration order [hello world]", all[0](), all[1]())
	}
}

// 锁定单值 nil 产物:provider 已执行但返回 nil 指针时,
// 产物不会进入缓存,注入按"依赖缺失"报错,而不是注入 nil。
func TestPatternNilSingleValueIsMissing(t *testing.T) {
	di := New()

	type dep struct{}
	mustProvideLock(t, di, func() *dep { return nil })

	err := di.TryInject(func(d *dep) {})
	if err == nil {
		t.Fatal("inject with nil provider output should fail")
	}
	if !strings.Contains(err.Error(), "value not found") {
		t.Fatalf("error should report missing value, got: %v", err)
	}
}

// 锁定 example/list:列表注入按 provider 注册顺序聚合,
// provider 内部返回的切片元素顺序保持不变。
func TestPatternListAggregationOrder(t *testing.T) {
	di := New()

	type Middleware func(string) string

	mustProvideLock(t, di, func() []Middleware {
		return []Middleware{func(s string) string { return "[auth] " + s }}
	})
	mustProvideLock(t, di, func() []Middleware {
		return []Middleware{
			func(s string) string { return "[log] " + s },
			func(s string) string { return "[trace] " + s },
		}
	})

	var chain []Middleware
	var latest Middleware
	if err := di.TryInject(func(all []Middleware, last Middleware) { chain, latest = all, last }); err != nil {
		t.Fatalf("TryInject: %v", err)
	}

	if len(chain) != 3 {
		t.Fatalf("chain size = %d, want 3", len(chain))
	}
	for i, want := range []string{"[auth] x", "[log] x", "[trace] x"} {
		if got := chain[i]("x"); got != want {
			t.Fatalf("chain[%d] = %q, want %q", i, got, want)
		}
	}
	if latest("x") != "[trace] x" {
		t.Fatalf("latest = %q, want the last registered provider", latest("x"))
	}
}

// 锁定 example/map:
//  1. 多个 provider 可向同一个 map[string]T 贡献不同 key;
//  2. 相同 key 由后注册的 provider 覆盖;
//  3. provider 返回 map 中的 nil 值被跳过;
//  4. 空字符串 key 归入 default 分组,注入后表现为 key "default"。
func TestPatternMapNamespaceAggregation(t *testing.T) {
	di := New()

	type DB struct{ DSN string }

	mustProvideLock(t, di, func() map[string]*DB {
		return map[string]*DB{
			"master": {DSN: "master"},
			"dup":    {DSN: "first"},
			"nilkey": nil,
		}
	})
	mustProvideLock(t, di, func() map[string]*DB {
		return map[string]*DB{
			"analytics": {DSN: "analytics"},
			"dup":       {DSN: "second"},
			"":          {DSN: "empty-key"},
		}
	})

	var dbs map[string]*DB
	if err := di.TryInject(func(m map[string]*DB) { dbs = m }); err != nil {
		t.Fatalf("TryInject: %v", err)
	}

	if dbs["master"] == nil || dbs["analytics"] == nil {
		t.Fatalf("namespaced keys missing, map = %v", dbs)
	}
	if dbs["dup"].DSN != "second" {
		t.Fatalf("duplicate key = %q, want last registered provider's %q", dbs["dup"].DSN, "second")
	}
	if _, ok := dbs["nilkey"]; ok {
		t.Fatal("nil values in provider map must be skipped")
	}
	if dbs["default"] == nil || dbs["default"].DSN != "empty-key" {
		t.Fatalf(`empty map key must surface as "default", got %v`, dbs["default"])
	}
}

// 锁定 example/inject-method:
//  1. DixInject 前缀方法在结构体字段注入之前执行(此时字段仍为零值);
//  2. 方法按名称字母序依次注入;
//  3. 方法参数解析规则与函数注入一致:单值取最后注册者,切片聚合全部。
func TestPatternMethodInjection(t *testing.T) {
	di := New()

	mustProvideLock(t, di, func() *patternMethodDep { return &patternMethodDep{N: 1} })
	mustProvideLock(t, di, func() error { return errors.New("primary") })
	mustProvideLock(t, di, func() error { return errors.New("secondary") })

	target := &patternMethodTarget{}
	if err := di.TryInject(target); err != nil {
		t.Fatalf("TryInject: %v", err)
	}

	if got, want := strings.Join(target.order, ","), "All,One"; got != want {
		t.Fatalf("method call order = %q, want %q (alphabetical)", got, want)
	}
	if target.sawDep {
		t.Fatal("DixInject methods must run before struct field injection")
	}
	if target.Dep == nil || target.Dep.N != 1 {
		t.Fatalf("field not injected after methods: %+v", target.Dep)
	}
	if target.single == nil || target.single.Error() != "secondary" {
		t.Fatalf("single value = %v, want last registered provider", target.single)
	}
	if len(target.all) != 2 || target.all[0].Error() != "primary" || target.all[1].Error() != "secondary" {
		t.Fatalf("aggregate = %v, want [primary secondary]", target.all)
	}
}

type patternMethodTarget struct {
	Dep    *patternMethodDep
	order  []string
	single error
	all    []error
	sawDep bool
}

type patternMethodDep struct{ N int }

func (t *patternMethodTarget) DixInjectAll(errs []error) {
	t.order = append(t.order, "All")
	t.all = errs
	t.sawDep = t.Dep != nil
}

func (t *patternMethodTarget) DixInjectOne(err error) {
	t.order = append(t.order, "One")
	t.single = err
}

// 锁定 example/provide-multi-output:provider 返回的 struct 按导出字段拆分注册,
// 各字段共享同一底层实例,字段内的嵌套依赖递归解析。
func TestPatternStructOutSharedInstance(t *testing.T) {
	di := New()

	type Config struct{ DSN string }
	type Database struct{ Config *Config }
	type UserService struct{ DB *Database }
	type OrderService struct{ DB *Database }
	type In struct{ Config *Config }
	type Out struct {
		DB       *Database
		UserSvc  *UserService
		OrderSvc *OrderService
	}

	mustProvideLock(t, di, func() *Config { return &Config{DSN: "dsn"} })
	mustProvideLock(t, di, func(in In) Out {
		db := &Database{Config: in.Config}
		return Out{DB: db, UserSvc: &UserService{DB: db}, OrderSvc: &OrderService{DB: db}}
	})

	var user *UserService
	var order *OrderService
	if err := di.TryInject(func(u *UserService, o *OrderService) { user, order = u, o }); err != nil {
		t.Fatalf("TryInject: %v", err)
	}

	if user == nil || order == nil {
		t.Fatalf("services not injected: user=%v order=%v", user, order)
	}
	if user.DB != order.DB {
		t.Fatal("fields of one provider output must share the same instance")
	}
	if user.DB.Config == nil || user.DB.Config.DSN != "dsn" {
		t.Fatalf("nested struct input not resolved: %+v", user.DB)
	}
}

// 锁定 example/singleton:同类型依赖在容器内是单例——
// 不同 provider、注入函数拿到的都是同一实例。
func TestPatternSingletonSharing(t *testing.T) {
	di := New()

	type Logger struct{ Name string }
	type Redis struct{ Log *Logger }
	type Cache struct{ Log *Logger }

	mustProvideLock(t, di, func() *Logger { return &Logger{Name: "app"} })
	mustProvideLock(t, di, func(l *Logger) *Redis { return &Redis{Log: l} })
	mustProvideLock(t, di, func(l *Logger) *Cache { return &Cache{Log: l} })

	var redis *Redis
	var cache *Cache
	var fnLog *Logger
	if err := di.TryInject(func(r *Redis, c *Cache, l *Logger) {
		redis, cache, fnLog = r, c, l
	}); err != nil {
		t.Fatalf("TryInject: %v", err)
	}

	if fnLog == nil || fnLog.Name != "app" {
		t.Fatalf("logger not resolved: %+v", fnLog)
	}
	if redis.Log != cache.Log || redis.Log != fnLog {
		t.Fatal("same-type dependencies must be singleton-shared across consumers")
	}
}

// 锁定 example/error-handling:
//  1. provider 返回非 nil error 时注入失败,原始错误可通过 errors.Is 判定;
//  2. 失败的 provider 不缓存,下次 Inject 重新执行(与超时 provider 永不重试相反);
//  3. 注入函数自身返回的 error 原样保留在错误链上。
func TestPatternProviderErrorPropagation(t *testing.T) {
	type DB struct{}

	sentinel := errors.New("boom")
	di := New()
	calls := 0
	mustProvideLock(t, di, func() (*DB, error) {
		calls++
		return nil, sentinel
	})

	err := di.TryInject(func(db *DB) error { return nil })
	if !errors.Is(err, sentinel) {
		t.Fatalf("error chain lost sentinel: %v", err)
	}
	if calls != 1 {
		t.Fatalf("provider calls = %d, want 1", calls)
	}

	if err := di.TryInject(func(db *DB) error { return nil }); !errors.Is(err, sentinel) {
		t.Fatalf("second inject: %v", err)
	}
	if calls != 2 {
		t.Fatalf("failed provider must re-execute on next inject, calls = %d", calls)
	}

	di2 := New()
	mustProvideLock(t, di2, func() *DB { return &DB{} })
	injectSentinel := errors.New("inject_err")
	err = di2.TryInject(func(db *DB) error { return injectSentinel })
	if !errors.Is(err, injectSentinel) {
		t.Fatalf("injected function error lost: %v", err)
	}
}

// 锁定 example/empty-collections:
//  1. 默认容器(AllowValuesNull=true)把缺失的 map/list 依赖解析为非 nil 空集合;
//  2. WithRejectEmptyCollections 后缺失的 map/list 依赖导致注入失败;
//  3. 单值依赖缺失永远失败,与 AllowValuesNull 无关。
func TestPatternNilCollectionsTolerance(t *testing.T) {
	type Handler func() string
	type app struct {
		Handlers []Handler
		Errors   map[string]error
	}

	tolerant := New()
	var okApp app
	if err := tolerant.TryInject(&okApp); err != nil {
		t.Fatalf("tolerant inject: %v", err)
	}
	if okApp.Handlers == nil || len(okApp.Handlers) != 0 {
		t.Fatalf("missing list must resolve to non-nil empty slice, got %#v", okApp.Handlers)
	}
	if okApp.Errors == nil || len(okApp.Errors) != 0 {
		t.Fatalf("missing map must resolve to non-nil empty map, got %#v", okApp.Errors)
	}

	reject := New(WithRejectEmptyCollections())
	var rejectedApp app
	if err := reject.TryInject(&rejectedApp); err == nil {
		t.Fatal("reject-empty-collections container must fail on missing collections")
	}

	type single struct{ Dep *Handler }
	if err := tolerant.TryInject(&single{}); err == nil {
		t.Fatal("missing single-value dependency must fail even with AllowValuesNull")
	}
}

// 锁定 example/inject-struct:注入目标结构体的导出指针字段递归解析,
// 未导出字段跳过且不报错。
func TestPatternStructInNestedResolution(t *testing.T) {
	di := New()

	type Config struct{ DSN string }
	type Database struct {
		Config *Config
		note   string
	}
	type Metadata struct{ Version string }
	type App struct {
		DB       *Database
		Metadata *Metadata
		note     string
	}

	mustProvideLock(t, di, func() *Config { return &Config{DSN: "app"} })
	mustProvideLock(t, di, func(c *Config) *Database { return &Database{Config: c} })
	mustProvideLock(t, di, func() *Metadata { return &Metadata{Version: "v1"} })

	app := &App{}
	if err := di.TryInject(app); err != nil {
		t.Fatalf("TryInject: %v", err)
	}
	if app.DB == nil || app.DB.Config == nil || app.DB.Config.DSN != "app" {
		t.Fatalf("nested resolution failed: %+v", app.DB)
	}
	if app.Metadata == nil || app.Metadata.Version != "v1" {
		t.Fatalf("metadata not injected: %+v", app.Metadata)
	}
}
