# [Unreleased]

> 推荐维护方式：
>
> - 建议通过 agent 提示词执行：`/changelog-maintenance draft|release`

## 新增

- 新增 `docs/pr_description_template.md`，统一 PR 描述结构（Summary / Changes / Test Plan / Risk / Checklist），提升评审与合并效率。

## 修复

- 修复 `dixglobal/global_test.go` 中 `TestInjectT` 的全局状态依赖问题，避免测试执行顺序敏感导致的偶发失败。

## 变更

- CI 流水线补充并稳定了 `go test ./... -race`、`golangci-lint` 与任务脚本协同，保持主干质量门禁一致。
- `Taskfile` 测试范围从仅核心包扩展为全仓库，减少漏测边缘模块的风险。

## 文档

- 重构中英文 `README`：补充 API/选项速查、线程安全说明、生产建议与诊断入口，提升新用户上手效率。
- 补充 `dixhttp` 反向代理鉴权示例与安全清单，强调内网部署与敏感诊断接口保护。
- 全面优化 `example/*` 示例：统一 `dix.New()` 用法、简化场景命名与输出，降低学习与复制成本。
