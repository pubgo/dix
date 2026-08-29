# [Unreleased]

> 推荐维护方式：
>
> - 建议通过 agent 提示词执行：`/changelog-maintenance draft|release`

## 新增

暂无

## 修复

- 修复 Inject/TryInject 调用级 Option 被容器默认值覆盖而失效的问题，调用方 Option 现优先于容器默认值 (#39)
- 修复 provider 调用超时后被静默重复执行的问题：超时的 provider 不再自动重试，后续注入返回明确错误 (#38)
- 修复 Provide/TryProvide 静默接受不支持的输入/输出类型的问题，注册阶段即返回错误 (#40)

## 变更

- 对齐 CI 与 Taskfile：example 模块纳入构建与测试，统一 -race/覆盖率(atomic) 与 lint 超时，忽略 coverage.html (#46)
- 移除 Options.Merge：其"容器值优先"的覆盖语义是调用级 Option 失效的根因 (#39)

## 文档

- 补充 ProviderTimeout 语义说明：超时调用不可中止，且失败后不会被重新执行 (#38)
