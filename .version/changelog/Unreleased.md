# [Unreleased]

> 推荐维护方式：
>
> - 建议通过 agent 提示词执行：`/changelog-maintenance draft|release`

## 新增

- 新增 dixinternal 模式锁测试（pattern_lock_test.go / api_lock_test.go，17 个），将 example 演示的 provider/inject 行为契约锁死；dixinternal 覆盖率 69.7% → 83.5%
- example 全部 13 个示例新增 main_test.go：演示逻辑抽取为可返回结果的函数，测试逐一断言各示例特性（含 http 端到端装配）

## 修复

暂无

## 变更

暂无

## 文档

- example 注释统一中文模板（功能/原理/运行/预期输出），实测核对预期输出，修正 lazy 示例与实际行为不符的注释
- 新增 example/README.md 索引：示例 ↔ 特性 ↔ 锁测试对照
