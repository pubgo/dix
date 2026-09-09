// 【功能】dixhttp 依赖图可视化:大规模端到端综合示例。
//
// 【原理】按微服务分层组织包(bootstrap / app / router / infra / plugins /
// domain/*/models|infra|logic|service|handler),构造真实项目规模的容器并
// 触发多类可诊断错误,用于验证 legacy `/` UI。
//
// 【运行】
//
//	cd example/http && go run .
//	# 或:task web-demo
//
// 【可选环境变量】
//
//	DIX_HTTP_ADDR=:8080
//	DIX_TRACE_DI=true
//	DIX_DIAG_FILE=.local/dix-diag.jsonl
package main

import (
	"log"

	"github.com/pubgo/dix/v2"

	"github.com/pubgo/dix/example/http/bootstrap"
)

// buildContainer 测试入口,转发到 bootstrap.Build。
func buildContainer() *dix.Dix {
	return bootstrap.Build()
}

func main() {
	if err := bootstrap.Run(); err != nil {
		log.Fatal("Server error:", err)
	}
}
