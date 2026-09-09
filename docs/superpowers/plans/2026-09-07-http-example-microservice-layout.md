# HTTP Example Microservice Layout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure `example/http` into bootstrap + app + router + infra + plugins(interface/impls) + domain vertical slices with per-package Provide.

**Architecture:** Each leaf package owns types + `Provide(*dix.Dix)`. `bootstrap.Build` is the only assembler. Plugins expose interfaces; impls contribute via `map[string]T` namespaces.

**Tech Stack:** Go, dix/v2, dixhttp, existing `example/http` tests.

## Global Constraints

- No domain-root `provide.go` / `Providers()` facade
- Provide only inside owning packages
- Three pyramid entries: Application, ScaleFixture, TimeoutProbe
- ≥190 providers, ≥10 modules
- Spec: `docs/superpowers/specs/2026-09-07-http-example-microservice-layout-design.md`

---

### Task 1: Domain layer packages (10 domains × 5 layers)

**Files:**
- Create: `example/http/domain/<d>/{models,infra,logic,service,handler}/*.go` for each of: analytics, billing, identity, inventory, media, notification, searchx, shipping, storage, workflow
- Delete: `example/http/domain/<d>/<d>.go`

**Interfaces:**
- Produces per domain: `Config`, `Client`, `Regions`, `Repo`, `Service`, `Handler` with same dependency chain as today’s flat package; each file exports `func Provide(di *dix.Dix)`.

- [ ] **Step 1: Generate domain packages**

Use a generator (inline Go or shell) so every domain matches billing:

```go
// domain/billing/models/config.go
package models

import "github.com/pubgo/dix/v2"

type Config struct {
	Env     string
	Timeout string
}

func Provide(di *dix.Dix) {
	dix.Provide(di, func() *Config {
		return &Config{Env: "prod", Timeout: "3s"}
	})
}
```

```go
// domain/billing/infra/client.go
package infra

import (
	"github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/example/http/domain/billing/models"
)

type Client struct{ Config *models.Config }
type Regions map[string]*Client

func Provide(di *dix.Dix) {
	dix.Provide(di, func(c *models.Config) *Client { return &Client{Config: c} })
	dix.Provide(di, func(c *models.Config) Regions {
		return Regions{"cn": {Config: c}, "us": {Config: c}, "eu": {Config: c}}
	})
}
```

```go
// domain/billing/logic/repo.go
package logic

import (
	"github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/example/http/domain/billing/infra"
)

type Repo struct{ Client *infra.Client }

func Provide(di *dix.Dix) {
	dix.Provide(di, func(c *infra.Client) *Repo { return &Repo{Client: c} })
}
```

```go
// domain/billing/service/service.go
package service

import (
	"github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/example/http/domain/billing/logic"
)

type Service struct{ Repo *logic.Repo }

func Provide(di *dix.Dix) {
	dix.Provide(di, func(r *logic.Repo) *Service { return &Service{Repo: r} })
}
```

```go
// domain/billing/handler/handler.go
package handler

import (
	"github.com/pubgo/dix/v2"
	binfra "github.com/pubgo/dix/example/http/domain/billing/infra"
	"github.com/pubgo/dix/example/http/domain/billing/service"
)

type Handler struct{ Service *service.Service }

func Provide(di *dix.Dix) {
	dix.Provide(di, func(s *service.Service, regions binfra.Regions) *Handler {
		_ = regions
		return &Handler{Service: s}
	})
}
```

Repeat for all 10 domains (only import path / comments change). Delete old flat files.

- [ ] **Step 2: Compile-check domains**

Run: `cd example/http && go build ./domain/...`  
Expected: success

---

### Task 2: infra + plugins + app + router

**Files:**
- Create: `example/http/infra/logger/logger.go`
- Create: `example/http/infra/diag/diag.go`
- Create: `example/http/infra/scale/scale.go` (enough distinct Provide calls to keep ≥190 total)
- Create: `example/http/plugins/plugin.go` (interfaces + Platform Provide)
- Create: `example/http/plugins/<name>/*.go` for ~20 impls (auth, billing, cache, … vault) each providing map namespaces
- Create: `example/http/app/application.go`
- Create: `example/http/router/server.go`

**Interfaces:**
- `plugins.Plugin` with `Name() string`; `plugins.Worker` with `Name() string`; `plugins.Platform` with `Names []string`
- Each impl: `Provide` returns `map[string]plugins.Plugin` and `map[string]plugins.Worker` depending on Plugin map entry
- Platform Provide consumes `map[string]plugins.Worker`

- [ ] **Step 1: Implement logger, diag, scale, plugins, app, router as above**

Scale: register ≥100 tiny named provider types OR ≥100 single-key map contributions via separate `Provide` funcs so total container providers ≥190 after domains+plugins.

- [ ] **Step 2: `go build ./infra/... ./plugins/... ./app/... ./router/...`**

Expected: success

---

### Task 3: bootstrap + thin main + tests

**Files:**
- Create: `example/http/bootstrap/container.go`, `run.go`
- Modify: `example/http/main.go` (thin)
- Delete: `example/http/plugins.go`
- Modify: `example/http/main_test.go`, `scale_shape_test.go`

**Interfaces:**
- `bootstrap.Build() *dix.Dix`
- `bootstrap.Run() error`
- Tests call `bootstrap.Build()` or `main` wrapper `buildContainer()` → `bootstrap.Build()`

- [ ] **Step 1: Wire bootstrap.Build calling all Provide in order**
- [ ] **Step 2: Update tests** — plugin count = `len(platform.Names)` expected (2 × impl count if plugin+worker names); pyramid still 3 entries
- [ ] **Step 3: `go test ./example/http/...`**

Expected: PASS

- [ ] **Step 4: Commit** (only if user asks)
