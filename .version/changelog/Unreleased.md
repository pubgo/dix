# [Unreleased]

> 推荐维护方式：
>
> - 建议通过 agent 提示词执行：`/changelog-maintenance draft|release`

## 新增

- 新增 dixinternal 模式锁测试（pattern_lock_test.go / api_lock_test.go，17 个），将 example 演示的 provider/inject 行为契约锁死；dixinternal 覆盖率 69.7% → 83.5%

## 修复

暂无

## 变更

暂无

## 文档

- example 注释统一中文模板（功能/原理/运行/预期输出），实测核对预期输出，修正 lazy 示例与实际行为不符的注释
- 新增 example/README.md 索引：示例 ↔ 特性 ↔ 锁测试对照
