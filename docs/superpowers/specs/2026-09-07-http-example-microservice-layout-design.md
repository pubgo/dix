# HTTP Example Microservice Layout Design

**Date:** 2026-09-07  
**Status:** Approved  
**Scope:** Restructure `example/http` into a clear microservice-style Go package layout so ModuleGraph / 模块地图 reflects real system boundaries (not visualization virtual buckets).

## Goal

Organize the dixhttp scale demo as a complete small system:

- Vertical domain slices with real packages: `models` → `infra` → `logic` → `service` → `handler`
- Shared shells: `app`, `router`, `infra/*`, `plugins`, `bootstrap`
- Each package registers its own `Provide(di)` — **no** domain-root aggregator `provide.go`
- `bootstrap` is the only DI assembly / startup orchestration entry
- Plugins: public **interfaces**, several **implementations**, each with a Provider, consumers see interfaces only

## Non-goals

- Changing dixhttp UI semantics (providers pyramid, module map algorithm) beyond what falls out of real packages
- Keeping the old `Plugin[T]` / 60-role generic sea as the primary plugin model
- Domain-level `Providers()` / `provide.go` facades

## Package tree

```text
example/http/
  main.go                 # thin: bootstrap.Run()
  bootstrap/
    container.go          # Build() — call every module Provide in order
    run.go                # pre-create, startup diagnostics, hand off to router
  app/
    application.go        # Application aggregate + Provide
  router/
    server.go             # dixhttp listen / serve
  infra/
    logger/               # Logger interface + ConsoleLogger Provide
    diag/                 # SlowRemote / TimeoutProbe + startup demo types
  plugins/
    plugin.go             # Plugin / Worker / Platform interfaces + Platform Provide
    auth/                 # impl + Provide → map[string]plugins.Plugin (and Worker)
    billing/
    cache/
    …                     # several impl packages (not one mega main package)
  domain/
    billing/
      models/             # Config + Provide
      infra/              # Client, Regions + Provide
      logic/              # Repo + Provide
      service/            # Service + Provide
      handler/            # Handler + Provide
    analytics|identity|inventory|media|notification|searchx|shipping|storage|workflow/
      (same five layers)
```

## Rules

1. **Provide lives in the owning package.** Bootstrap imports packages and calls `Provide(*dix.Dix)`; it does not define domain providers.
2. **No domain-root `provide.go`.** There is no `domain/billing.Providers`.
3. **Import paths are the module identity.** Prefer short package names (`models`, `handler`, …) under each domain path; bootstrap uses import aliases when needed.
4. **Plugins**
   - Contract package `plugins` exports `Plugin`, `Worker`, and `Platform` (or equivalent).
   - Each implementation package provides `map[string]plugins.Plugin` / `map[string]plugins.Worker` (dix namespace merge) so multiple impls coexist.
   - `Platform` depends on `map[string]plugins.Worker` (or Plugin) and exposes names/counts to `Application`.
   - `Application` depends on `*plugins.Platform` and domain handlers/services — not concrete plugin impl types.
5. **Pyramid entries:** `*app.Application` and `*diag.TimeoutProbe` (no artificial ScaleFixture).
6. **Scale comes from real packages** (domain layers + plugin impls), not pad providers.

## Wiring order (bootstrap)

1. `infra/logger`
2. All `domain/*/models|infra|logic|service|handler` (models before infra before logic before service before handler; domains independent)
3. All `plugins/<impl>` then `plugins` Platform
4. `app` Application
5. `infra/diag` timeout chain (entry #2)

`main` only calls `bootstrap.Run()`.

## Test contracts

| Test | Expectation after change |
|------|---------------------------|
| `TestBuildContainerWiresApplication` | Ten domain services wired; plugin name count = number of registered plugin/worker namespaces (not 120) |
| `TestDemoContainerShape` | ≥10 modules; provider/object floors from real domain+plugins |
| `TestPyramidHasTwoBusinessEntries` | Exactly two unconsumed business entry providers (Application, TimeoutProbe) |

## Migration

1. Add new packages; generate ten identical domain layer trees from the former flat `domain/<name>/<name>.go`.
2. Replace `plugins.go` with interface + impl packages.
3. Move Application / Logger / Scale / Timeout / server into `app` / `infra` / `router`.
4. Point tests at `bootstrap.Build()` (or keep `buildContainer` as a one-liner wrapper in main for minimal test churn).
5. Delete old flat domain files and `plugins.go` bodies in `main`.
6. Optionally simplify module-map virtual coarse buckets once real packages disperse providers.

## Success criteria

- `go test ./example/http/...` passes
- ModuleGraph paths show `domain/.../handler`, `plugins/auth`, `bootstrap` not used as a provider home
- Browser 模块地图 shows deep real packages without relying on visualization-only splits for domains
