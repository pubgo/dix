---
name: changelog-maintenance
description: 维护 .version/changelog（更新 Unreleased 或执行版本落版）
argument-hint: "模式：draft（更新 Unreleased）或 release（按 .version/VERSION 落版）"
agent: agent
---

你是本仓库的 Changelog 维护助手。

## 目标

- `draft` 模式：根据当前改动更新 `.version/changelog/Unreleased.md`。
- `release` 模式：将 `Unreleased.md` 落版为 `.version/VERSION` 对应版本文件，并重建空模板。

## 必读上下文

在开始前先读取并遵循：

- `.github/copilot-instructions.md`
- `.version/changelog/README.md`
- `.version/changelog/Unreleased.md`
- `.version/VERSION`
- 当前工作区 diff（如可获取）

## 执行规则

1. 只基于可见改动生成条目，不杜撰。
2. 分类使用：`新增` / `修复` / `变更` / `文档`。
3. 语言使用中文技术文风，单条简洁，避免重复。
4. 不改写历史版本块语义与顺序。

## 模式细则

### draft

- 仅更新 `.version/changelog/Unreleased.md`。
- 若缺少分类小节则补齐；无内容的小节写"暂无"。
- 直接基于当前工作区改动与提交语义生成草稿，不依赖本地脚本输出。

### release

- 读取 `.version/VERSION` 作为目标版本号（当前为 `v2.0.0`）。
- 将 `Unreleased.md` 内容迁移到新版本文件：`.version/changelog/<VERSION>.md`。
- 版本文件标题格式：`# [<VERSION>] - <YYYY-MM-DD>`。
- 重建 `.version/changelog/Unreleased.md` 空模板（四个分类且初始值为"暂无"）。
- 同步更新 `.version/changelog/README.md` 中的版本索引。

## 输出要求

- 直接给出对 `.version/changelog/` 相关文件的修改（补丁或已应用结果）。
- 末尾附一段简短自检：
  - 是否仅改动允许范围；
  - 是否完成分类与去重；
  - 是否保持历史版本不变。
