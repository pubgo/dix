package dixinternal

// 第二批锁测试:容器自省 API(dixhttp 可视化的数据源)、
// provider panic 恢复、诊断文件读取、注册校验等核心路径的行为契约。

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type apiLockSvc struct{ N int }

type apiLockInner struct {
	Svc *apiLockSvc
}

type apiLockPayload struct {
	Svc   *apiLockSvc
	Named map[string]*apiLockSvc
	Inner apiLockInner
	Count int // 基础类型:不作为依赖
}

// 锁定 provider panic 行为(对应 http 示例的 provider_panic 场景):
//  1. provider 内 panic 被恢复,注入以 error 失败,panic 消息保留;
//  2. 失败记录进入 recentErrors,错误类型为 provider_panic;
//  3. panic 的 provider 不缓存,下次注入重新执行。
func TestPatternProviderPanicRecovered(t *testing.T) {
	di := New()
	calls := 0
	mustProvideLock(t, di, func() *apiLockSvc {
		calls++
		panic("boom-panic")
	})

	err := di.TryInject(func(s *apiLockSvc) {})
	if err == nil || !strings.Contains(err.Error(), "boom-panic") {
		t.Fatalf("panic should surface as error with original message, got %v", err)
	}

	var found *RecentError
	for _, rec := range di.GetRecentErrors(0) {
		if rec.ErrorType == "provider_panic" {
			found = &rec
			break
		}
	}
	if found == nil {
		t.Fatalf("provider_panic record expected in recent errors, got %+v", di.GetRecentErrors(0))
	}
	if !strings.Contains(found.RootCause, "boom-panic") {
		t.Fatalf("panic record should keep root cause, got %+v", found)
	}

	if err := di.TryInject(func(s *apiLockSvc) {}); err == nil {
		t.Fatal("second inject should re-run panicking provider and fail again")
	}
	if calls != 2 {
		t.Fatalf("panicking provider must re-execute, calls = %d", calls)
	}
}

// 锁定容器自省 API:这些接口是 dixhttp 可视化的数据源。
//   - GetProviders:输出类型 -> provider 列表(注册顺序);
//   - GetObjects:输出类型 -> 分组产物缓存,单例语义可见;
//   - GetProviderDetails:函数名、源码位置、输入/输出类型齐全;
//   - GetProviderRuntimeStats:执行次数与耗时统计。
func TestPatternContainerIntrospection(t *testing.T) {
	di := New()
	mustProvideLock(t, di, func() *apiLockSvc { return &apiLockSvc{N: 7} })
	// 带结构体输入的 provider:用于锁定输入类型的扁平化展示。
	mustProvideLock(t, di, func(in apiLockPayload) *apiLockInner { return &apiLockInner{} })

	var got *apiLockSvc
	if err := di.TryInject(func(s *apiLockSvc) { got = s }); err != nil {
		t.Fatalf("TryInject: %v", err)
	}

	svcType := reflect.TypeOf(&apiLockSvc{})
	providers := di.GetProviders()
	if len(providers[svcType]) != 1 {
		t.Fatalf("GetProviders should list one provider for %s, got %d", svcType, len(providers[svcType]))
	}

	objects := di.GetObjects()
	values := objects[svcType]["default"]
	if len(values) != 1 || values[0].Interface() != got {
		t.Fatalf("GetObjects should cache the singleton, got %v", values)
	}

	details := di.GetProviderDetails()
	var detail *ProviderDetails
	for i := range details {
		if details[i].OutputType == svcType.String() {
			detail = &details[i]
			break
		}
	}
	if detail == nil {
		t.Fatalf("GetProviderDetails should describe the %s provider, got %+v", svcType, details)
	}
	if detail.FunctionName == "" || detail.FunctionFile == "" || detail.FunctionLine <= 0 {
		t.Fatalf("provider source location missing: %+v", detail)
	}
	if detail.OutputPkg != "github.com/pubgo/dix/v2/dixinternal" {
		t.Fatalf("unexpected output package %q", detail.OutputPkg)
	}

	var innerDetail *ProviderDetails
	innerType := reflect.TypeOf(&apiLockInner{})
	for i := range details {
		if details[i].OutputType == innerType.String() {
			innerDetail = &details[i]
			break
		}
	}
	if innerDetail == nil {
		t.Fatalf("GetProviderDetails should describe the %s provider", innerType)
	}
	// 结构体输入扁平化并去重后只剩 *apiLockSvc;基础类型字段 Count 不出现。
	if len(innerDetail.InputTypes) != 1 || innerDetail.InputTypes[0] != svcType.String() {
		t.Fatalf("struct input should flatten to [%s], got %v", svcType, innerDetail.InputTypes)
	}
	if len(innerDetail.InputPkgs) != 1 || innerDetail.InputPkgs[0] != "github.com/pubgo/dix/v2/dixinternal" {
		t.Fatalf("input packages should resolve, got %v", innerDetail.InputPkgs)
	}

	var stat *ProviderRuntimeStats
	for i := range di.GetProviderRuntimeStats() {
		s := di.GetProviderRuntimeStats()[i]
		if s.OutputType == svcType.String() {
			stat = &s
			break
		}
	}
	if stat == nil || stat.CallCount != 1 || stat.LastRunAtUnixNano <= 0 {
		t.Fatalf("runtime stats should record exactly one call, got %+v", stat)
	}
}

// 锁定 GetProvideAllInputTypes:provider 输入类型的扁平化规则——
// 指针/接口/函数直接保留;struct 递归展开导出字段;
// 基础类型字段跳过;map/slice 展开为元素类型。
func TestGetProvideAllInputTypesFlattening(t *testing.T) {
	got := GetProvideAllInputTypes(reflect.TypeOf(apiLockPayload{}))

	svcPtr := reflect.TypeOf(&apiLockSvc{})
	if len(got) != 3 {
		t.Fatalf("flattened inputs = %d, want 3 (field, map elem, nested field), got %v", len(got), got)
	}
	for i, typ := range got {
		if typ != svcPtr {
			t.Fatalf("flattened[%d] = %s, want %s", i, typ, svcPtr)
		}
	}
}

// 锁定容器构造校验:非法 Option(负超时)在 New 时直接 panic,fail-fast。
func TestNewDixRejectsInvalidOptions(t *testing.T) {
	assertPanicsLock(t, func() { New(WithProviderTimeout(-time.Second)) })
	assertPanicsLock(t, func() { New(WithSlowProviderThreshold(-time.Millisecond)) })
}

func assertPanicsLock(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}

// 锁定 DIX_DIAG_FILE 读取契约:
//  1. 未配置环境变量时 Enabled=false,不报错;
//  2. 文件按 JSONL 解析,坏行跳过;缺 record_id 的记录回退为行号;
//  3. 结果按时间倒序,支持 kind 过滤与 Limit 分页(NextBefore)。
func TestReadDiagFileRecordsContract(t *testing.T) {
	t.Setenv(diagFileEnv, "")

	disabled, err := ReadDiagFileRecords(DiagFileQuery{Limit: 10})
	if err != nil {
		t.Fatalf("read with no env: %v", err)
	}
	if disabled.Enabled {
		t.Fatal("query must be disabled without DIX_DIAG_FILE")
	}

	path := filepath.Join(t.TempDir(), "diag.jsonl")
	lines := strings.Join([]string{
		`{"kind":"trace","event":"resolve.start","occurred_at_unix_nano":100}`,
		`{not-json`,
		`{"kind":"error","occurred_at_unix_nano":200,"fields":{"record_id":42}}`,
		``,
	}, "\n")
	if err := os.WriteFile(path, []byte(lines), 0o600); err != nil {
		t.Fatalf("write diag file: %v", err)
	}
	t.Setenv(diagFileEnv, path)

	result, err := ReadDiagFileRecords(DiagFileQuery{Limit: 10})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !result.Enabled || !result.Exists || result.Total != 2 {
		t.Fatalf("unexpected result header: enabled=%v exists=%v total=%d", result.Enabled, result.Exists, result.Total)
	}

	// 时间倒序:error(200, id=42)在前;trace 回退行号 1。
	first, second := result.Records[0], result.Records[1]
	if first.Kind != "error" || first.RecordID != 42 {
		t.Fatalf("first record = %+v, want error with id 42", first)
	}
	if second.Kind != "trace" || second.RecordID != 1 {
		t.Fatalf("second record = %+v, want trace with fallback line id 1", second)
	}

	filtered, err := ReadDiagFileRecords(DiagFileQuery{Kind: "error"})
	if err != nil || filtered.Total != 1 || filtered.Records[0].Kind != "error" {
		t.Fatalf("kind filter failed: %+v err=%v", filtered, err)
	}

	paged, err := ReadDiagFileRecords(DiagFileQuery{Limit: 1})
	if err != nil || paged.Returned != 1 || paged.NextBefore != 42 {
		t.Fatalf("pagination failed: returned=%d nextBefore=%d err=%v", paged.Returned, paged.NextBefore, err)
	}
}

// 锁定位置解析工具:GetFnName/GetFnTraceName 对非 main 函数输出全名,
// parseModulePath/inferPkgPathFromFile 从 go.mod 推导包路径。
func TestLocationHelpers(t *testing.T) {
	fn := func() {}
	fnVal := reflect.ValueOf(fn)

	if name := GetFnName(fnVal); !strings.Contains(name, "TestLocationHelpers") {
		t.Fatalf("GetFnName = %q, want closure in this test", name)
	}
	if name := GetFnTraceName(fnVal); name != GetFnName(fnVal) {
		t.Fatalf("GetFnTraceName should pass through non-main names, got %q vs %q", name, GetFnName(fnVal))
	}

	if file, line := resolveFuncLocation(fnVal); file == "" || line <= 0 {
		t.Fatalf("resolveFuncLocation = %s:%d, want real location", file, line)
	}
	if got := resolveFuncPkgPath("github.com/pubgo/dix/v2/dixinternal.New"); got != "github.com/pubgo/dix/v2/dixinternal" {
		t.Fatalf("resolveFuncPkgPath = %q", got)
	}
	if got := resolveFuncPkgPath("nofile"); got != "" {
		t.Fatalf("resolveFuncPkgPath without dot = %q, want empty", got)
	}
	if got := resolveTypePkgPath(reflect.TypeOf(&apiLockSvc{})); got != "github.com/pubgo/dix/v2/dixinternal" {
		t.Fatalf("resolveTypePkgPath = %q", got)
	}

	if got := parseModulePath("module github.com/pubgo/dix/v2\n\ngo 1.24\n"); got != "github.com/pubgo/dix/v2" {
		t.Fatalf("parseModulePath = %q", got)
	}
	if got := parseModulePath("invalid"); got != "" {
		t.Fatalf("parseModulePath on junk = %q, want empty", got)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/m\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if got := inferPkgPathFromFile(filepath.Join(dir, "sub", "main.go")); got != "example.com/m/sub" {
		t.Fatalf("inferPkgPathFromFile = %q, want example.com/m/sub", got)
	}
	if got := inferPkgPathFromFile(""); got != "" {
		t.Fatalf("inferPkgPathFromFile empty input = %q", got)
	}
}

// 锁定 normalizeRecoveredError 的三种归一化分支。
func TestNormalizeRecoveredError(t *testing.T) {
	sentinel := errors.New("real-error")
	if got := normalizeRecoveredError(sentinel); got != sentinel {
		t.Fatal("error values must pass through unchanged")
	}
	if got := normalizeRecoveredError("str-panic"); got.Error() != "str-panic" {
		t.Fatalf("string panic = %q", got.Error())
	}
	if got := normalizeRecoveredError(123); got.Error() != "panic: 123" {
		t.Fatalf("other panic = %q", got.Error())
	}
	if got := normalizeRecoveredError(nil); got == nil {
		t.Fatal("nil panic must still yield an error")
	}
}
