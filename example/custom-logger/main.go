// 【功能】自定义日志:用 SetLog 接管 dix 的内部日志输出。
//
// 【原理】dix 通过 slog 输出诊断日志(provider 缺失、超时、注入失败等):
//   - 默认写到 stderr;
//   - dix.SetLog(handler) 可将其接入应用的日志体系(自定义 handler/格式/落点);
//   - 本例注入一个缺失的依赖,触发 "provider not found" 与 "try inject failed"
//     两条告警,演示日志被自定义 handler 逐行收集。
//
// 该语义由 dixinternal 的 TestSetLog
// 与本目录 main_test.go 的 TestCollectDixLogs 锁定。
//
// 【运行】
//
//	cd example/custom-logger && go run .
//
// 【预期输出】(日志含时间戳与属性,此处省略)
//
//	captured 2 dix log record(s)
//	  - provider not found, please check imports or type definition
//	  - try inject failed
package main

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/pubgo/dix/v2"
)

func main() {
	records := collectDixLogs()

	fmt.Printf("captured %d dix log record(s)\n", len(records))
	for _, line := range records {
		fmt.Printf("  - %s\n", firstField(line, "msg="))
	}
}

// collectDixLogs 安装一个把日志行收进切片的自定义 handler,
// 然后触发一次"注入缺失依赖",返回收集到的完整日志行。
func collectDixLogs() []string {
	var records []string
	dix.SetLog(slog.NewTextHandler(&lineWriter{lines: &records}, nil))

	di := dix.New()
	_ = di.TryInject(func(missing *CustomDependency) {})
	return records
}

// CustomDependency 未注册任何 provider,用于触发告警日志。
type CustomDependency struct{}

// lineWriter 把每条 slog 记录的整行文本追加进切片,模拟日志采集端。
type lineWriter struct{ lines *[]string }

func (w *lineWriter) Write(p []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimSpace(string(p)), "\n") {
		if line != "" {
			*w.lines = append(*w.lines, line)
		}
	}
	return len(p), nil
}

// firstField 从 slog TextHandler 行中取出指定字段的值(去引号)。
func firstField(line, key string) string {
	idx := strings.Index(line, key)
	if idx < 0 {
		return line
	}
	rest := line[idx+len(key):]
	if sp := strings.IndexByte(rest, ' '); sp >= 0 {
		rest = rest[:sp]
	}
	return strings.Trim(rest, `"`)
}
