---
description: "Use when handling pull request review, code review feedback, or PR comment resolution tasks. In review mode, provide analysis and comments only, and do not modify repository files directly."
name: "PR Review Read-Only Mode"
---
# Pull Request Review: Read-Only Execution

- When the user asks to review a pull request, review comments, or code diffs, stay in **review-only mode**.
- Do **not** create, edit, rename, or delete repository files in this mode.
- Focus on:
  - identifying issues and risks,
  - suggesting concrete fixes,
  - proposing patch snippets in chat only when explicitly requested.
- If the user explicitly switches from review to implementation (for example: "please apply the fixes now"), confirm intent once and then proceed with edits.
