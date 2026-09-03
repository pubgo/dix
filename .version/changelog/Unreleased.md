# [Unreleased]

> 推荐维护方式：
>
> - 建议通过 agent 提示词执行：`/changelog-maintenance draft|release`

## 新增

- 新增 dixinternal 模式锁测试（pattern_lock_test.go / api_lock_test.go），将 example 演示的 provider/inject 行为契约锁死
- example 覆盖全部公开特性：新增 inject-generic / context-inject / inject-map-list / timeout / global / context-container / custom-logger 七个示例，每个示例均配套 main_test.go
- 补齐测试盲区：慢 provider 告警、根包公开 API 包装器（Inject/Provide/InjectContext/InjectT/Option）、dixhttp 的 stats/packages/trace/group-rules/index handler

## 修复

暂无

## 变更

- example 目录重构命名：按演示特性统一为 kebab-case（inject-* / provide-* / context-* 前缀），map-nil 与 list-nil 合并为 empty-collections；同步双语 README 示例索引

## 文档

- example 注释统一中文模板（功能/原理/运行/预期输出），实测核对预期输出，修正 lazy 示例与实际行为不符的注释
- 新增 example/README.md 索引：示例 ↔ 特性 ↔ 锁测试对照
