// Package example 收录 dix 的可运行示例,即"代码即文档"。
//
// 每个子目录是一个独立演示,均可用 `go run .` 直接运行:
//
//	func/               函数注入:同类型多 provider 的单值/列表语义
//	struct-in/          结构体注入:字段递归解析
//	struct-out/         结构体多输出:一个 provider 提供多个依赖
//	map/                Map 命名空间注入:按 key 分组
//	list/               列表聚合注入:按注册顺序聚合
//	lazy/               惰性求值:provider 仅在需要时执行
//	cycle/              循环依赖检测
//	map-nil/            缺失 map 依赖的容错(空集合)
//	list-nil/           缺失 slice 依赖的容错(空集合)
//	handler/            常见装配模式:单例共享 + 命名空间
//	inject_method/      DixInject 前缀方法注入
//	test-return-error/  TryProvide/TryInject 错误处理
//	http/               dixhttp 依赖图可视化(端到端大示例)
//
// 各示例演示的行为契约由 dixinternal/pattern_lock_test.go 锁定,
// 注释与实现不一致时以锁测试为准并修正注释。
package example
