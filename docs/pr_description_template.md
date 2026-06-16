# PR Description Template

Use this template to keep PRs review-friendly and merge-ready.

## Summary

- What changed at a high level (1-3 bullets).
- Why this change is needed.
- Any behavior or API changes reviewers should focus on.

## Changes

- **Core code**: key implementation updates.
- **Tests**: what was added/updated.
- **Docs/CI**: developer-facing updates.

## Test Plan

- [ ] `go test ./... -count=1 -race`
- [ ] `golangci-lint run --timeout=5m`

## Risk & Rollback

- Risk level: low / medium / high
- Main risk areas:
- Rollback plan:

## Checklist

- [ ] No unrelated files included
- [ ] Backward compatibility considered
- [ ] README/docs updated if behavior changed
