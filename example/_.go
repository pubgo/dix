// Package example 收录 dix 的可运行示例,即"代码即文档"。
//
// 每个子目录是一个独立演示,均可用 `go run .` 直接运行,命名按语义分组:
//
//	注入模式(inject-* / provide-*):
//	  inject-func/           函数类型注入
//	  inject-struct/         结构体字段注入(嵌套递归)
//	  inject-method/         DixInject 前缀方法注入
//	  inject-generic/        InjectT/InjectTContext 泛型注入
//	  context-inject/        InjectContext/TryInjectContext 与 trace 传播
//	  inject-map/            Map 命名空间注入
//	  inject-list/           列表聚合注入
//	  inject-map-list/       map[string][]T 分组聚合
//	  provide-multi-output/  结构体多输出 provider
//
//	运行语义:
//	  lazy/                  惰性求值与产物缓存
//	  cycle/                 循环依赖检测
//	  singleton/             容器级单例共享
//	  error-handling/        TryProvide/TryInject 错误处理
//	  timeout/               provider 超时与慢告警
//
//	容错:
//	  empty-collections/     缺失集合依赖解析为空集合(含 Reject 对照)
//
//	模块:
//	  global/                dixglobal 全局容器
//	  context-container/     dixcontext 容器随 Context 传递
//	  custom-logger/         SetLog 自定义日志
//	  http/                  dixhttp 依赖图可视化(端到端大示例)
//
// 各示例演示的行为契约由两层测试锁定:
//   - dixinternal/pattern_lock_test.go:核心容器语义;
//   - 各示例目录的 main_test.go:示例抽取出的演示函数本身。
//
// 注释与实现不一致时,以测试为准并修正注释。
package example
