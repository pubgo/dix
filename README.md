# dix

[![Go Reference](https://pkg.go.dev/badge/github.com/pubgo/dix/v2.svg)](https://pkg.go.dev/github.com/pubgo/dix/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/pubgo/dix)](https://goreportcard.com/report/github.com/pubgo/dix)

> **dix** is a lightweight yet powerful dependency injection framework for Go.

Inspired by [uber-go/dig](https://github.com/uber-go/dig), with support for advanced dependency management and namespace isolation.

[中文文档](./README_zh.md)

## Table of Contents

- [When to use dix](#when-to-use-dix)
- [Features](#-features)
- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [Core API](#-core-api)
- [Injection Patterns](#-injection-patterns)
- [Modules](#-modules)
- [Diagnostics](#-diagnostics)
- [Development](#️-development)
- [Examples](#-examples)
- [Documentation](#-documentation)

## When to use dix

- You need **runtime** dependency registration (plugins, dynamic modules, conditional wiring).
- You want **built-in diagnostics**: structured trace logs, JSONL export, and an HTTP dependency graph.
- You prefer a **dig-like API** with safe `Try*` variants, map/list grouping, and method injection.

For compile-time wiring with minimal runtime overhead, see [google/wire](https://github.com/google/wire). For Uber's fx ecosystem, see [uber-go/dig](https://github.com/uber-go/dig).

## ✨ Features

| Feature                  | Description                              |
| ------------------------ | ---------------------------------------- |
| 🔄 **Cycle Detection**    | Auto-detect dependency cycles            |
| 📦 **Multiple Injection** | Support func, struct, map, list          |
| 🏷️ **Namespace**          | Dependency isolation via map key         |
| 🎯 **Multi-Output**       | Struct can provide multiple dependencies |
| 🪆 **Nested Support**     | Support nested struct injection          |
| 🔧 **Non-Invasive**       | Zero intrusion to original objects       |
| 🛡️ **Safe API**           | `TryProvide`/`TryInject` won't panic     |
| 🌐 **Visualization**      | HTTP module for dependency graph         |

## 📦 Installation

```bash
go get github.com/pubgo/dix/v2
```

## 🚀 Quick Start

```go
package main

import (
    "fmt"
    "github.com/pubgo/dix/v2"
)

type Config struct {
    DSN string
}

type Database struct {
    Config *Config
}

type UserService struct {
    DB *Database
}

func main() {
    // Create container
    di := dix.New()

    // Register Providers
    dix.Provide(di, func() *Config {
        return &Config{DSN: "postgres://localhost/mydb"}
    })

    dix.Provide(di, func(c *Config) *Database {
        return &Database{Config: c}
    })

    dix.Provide(di, func(db *Database) *UserService {
        return &UserService{DB: db}
    })

    // Inject and use
    dix.Inject(di, func(svc *UserService) {
        fmt.Println("DSN:", svc.DB.Config.DSN)
    })
}
```

For production startup, prefer `TryProvide` / `TryInject` to avoid panics and keep the process alive for diagnostics:

```go
if err := dix.TryProvide(di, NewDatabase); err != nil {
    log.Fatal(err)
}
if err := dix.TryInject(di, Run); err != nil {
    log.Fatal(err)
}
```

## 📖 Core API

| API | Panics on error | Description |
| --- | --- | --- |
| `New(...Option)` | — | Create a container |
| `Provide(di, fn)` | yes | Register a provider |
| `TryProvide(di, fn)` | no | Register a provider, returns `error` |
| `Inject(di, target)` | yes | Inject into a function or struct |
| `TryInject(di, target)` | no | Inject, returns `error` |
| `InjectT[T](di)` | yes | Allocate a struct and inject exported fields |
| `InjectContext` / `TryInjectContext` | yes / no | Inject with trace context propagation |
| `Version()` | — | Return embedded version string |

Container options:

| Option | Default | Description |
| --- | --- | --- |
| `WithValuesNull()` | enabled | Allow nil provider results |
| `WithProviderTimeout(d)` | `15s` | Per-provider execution timeout (`0` = disabled) |
| `WithSlowProviderThreshold(d)` | `2s` | Warn when provider is slow (`0` = disabled) |

### Provide / TryProvide

Register constructor (Provider) to container:

```go
// Standard version - panics on error
dix.Provide(di, func() *Service { return &Service{} })

// Safe version - returns error
err := dix.TryProvide(di, func() *Service { return &Service{} })
if err != nil {
    log.Printf("Registration failed: %v", err)
}
```

### Inject / TryInject

Inject dependencies from container:

```go
// Function injection
dix.Inject(di, func(svc *Service) {
    svc.DoSomething()
})

// Struct injection
type App struct {
    Service *Service
    Config  *Config
}
app := &App{}
dix.Inject(di, app)

// Safe version
err := dix.TryInject(di, func(svc *Service) {
    // ...
})
```

### Generic Helpers

```go
// Inject into a new struct value
app := dix.InjectT[App](di)

// Inject with request-scoped trace context
err := dix.TryInjectContext(ctx, di, func(svc *Service) {
    svc.DoSomething()
})
```

### Thread Safety

`Dix` containers are **not thread-safe**. Do not call `Provide` / `Inject` (or their `Try*` variants) concurrently on the same container instance.

Recommended usage:

- Register all providers during application startup (single goroutine).
- After startup, only read resolved dependencies, or continue injection from a single goroutine.
- Use separate `Dix` instances per goroutine if you need isolated containers.
- For a process-wide singleton, prefer `dixglobal` only when startup is single-threaded.

### Startup Options

```go
di := dix.New(
    dix.WithProviderTimeout(2*time.Second),              // default: 15s; 0 disables
    dix.WithSlowProviderThreshold(300*time.Millisecond), // default: 2s; 0 disables
)
```

## 🎯 Injection Patterns

### Struct Injection

```go
type In struct {
    Config   *Config
    Database *Database
}

type Out struct {
    UserSvc  *UserService
    OrderSvc *OrderService
}

// Multiple inputs and outputs
dix.Provide(di, func(in In) Out {
    return Out{
        UserSvc:  &UserService{DB: in.Database},
        OrderSvc: &OrderService{DB: in.Database},
    }
})
```

### Map Injection (Namespace)

```go
// Provide with namespace
dix.Provide(di, func() map[string]*Database {
    return map[string]*Database{
        "master": &Database{DSN: "master-dsn"},
        "slave":  &Database{DSN: "slave-dsn"},
    }
})

// Inject specific namespace
dix.Inject(di, func(dbs map[string]*Database) {
    master := dbs["master"]
    slave := dbs["slave"]
})
```

### List Injection

```go
// Provide same type multiple times
dix.Provide(di, func() []Handler {
    return []Handler{&AuthHandler{}}
})
dix.Provide(di, func() []Handler {
    return []Handler{&LogHandler{}}
})

// Inject all
dix.Inject(di, func(handlers []Handler) {
    // handlers contains AuthHandler and LogHandler
})
```

## 🧩 Modules

### dixglobal - Global Container

Provides global singleton container for simple applications:

```go
import "github.com/pubgo/dix/v2/dixglobal"

// Use directly without creating container
dixglobal.Provide(func() *Config { return &Config{} })
dixglobal.Inject(func(c *Config) { /* ... */ })
```

### dixcontext - Context Integration

Bind container to `context.Context`:

```go
import "github.com/pubgo/dix/v2/dixcontext"

// Store in context
ctx := dixcontext.Create(context.Background(), di)

// Retrieve and use
container := dixcontext.Get(ctx)

// Non-panicking lookup
container = dixcontext.GetOrNil(ctx)
```

### dixhttp - Dependency Visualization

Web interface for visualizing dependency graphs, **designed for large projects**:

```go
import (
    "github.com/pubgo/dix/v2/dixhttp"
    "github.com/pubgo/dix/v2/dixinternal"
)

server := dixhttp.NewServer((*dixinternal.Dix)(di))
server.ListenAndServe(":8080")
```

Visit `http://localhost:8080` to view the dependency graph.

> **Security**: exposes dependency graphs, provider source locations, runtime errors, and trace data. Use on **localhost or private networks** only. Do not expose publicly without authentication.

**Highlights**:
- 🔍 **Fuzzy Search** - Quickly locate types or functions
- 📦 **Package Grouping** - Collapsible sidebar browsing
- 🔄 **Bidirectional Tracking** - Show both dependencies and dependents
- 📏 **Depth Control** - Limit display levels (1-5 or all)
- 🎨 **Modern UI** - Tailwind CSS + Alpine.js

See [dixhttp/README.md](./dixhttp/README.md) for API routes, event dictionary, and UI details.

## 🔍 Diagnostics

Optional observability for startup and injection troubleshooting. All file/console outputs are disabled unless configured.

| Env var | Default | Purpose |
| --- | --- | --- |
| `DIX_TRACE_DI` | off | Console step-by-step DI trace (`di_trace ...`) |
| `DIX_DIAG_FILE` | off | Append `trace` / `error` / `llm` records to JSONL |
| `DIX_TRACE_FILE` | off | Append trace-only JSONL (falls back to `DIX_DIAG_FILE`) |
| `DIX_LLM_DIAG_MODE` | `human` | Log mode: `human` / `machine` / `dual` |

```bash
export DIX_TRACE_DI=true
export DIX_DIAG_FILE=.local/dix-diag.jsonl
```

In-memory trace events (`dixtrace`) are enabled by default and queryable through `dixhttp` at `/api/trace`.

For the full `di_trace` event dictionary, HTTP APIs, and UI troubleshooting workflow, see [dixhttp/README.md](./dixhttp/README.md).

## 🛠️ Development

```bash
# Run all tests with coverage report
task test

# Lint and format
task lint

# go vet
task vet

# HTTP visualization demo
task web-demo
```

GitHub Actions runs `go test ./... -race` and `golangci-lint` on push/PR.

## 📚 Examples

| Example                                           | Description             |
| ------------------------------------------------- | ----------------------- |
| [struct-in](./example/struct-in/)                 | Struct input injection  |
| [struct-out](./example/struct-out/)               | Struct multi-output     |
| [func](./example/func/)                           | Function injection      |
| [map](./example/map/)                             | Map/namespace injection |
| [map-nil](./example/map-nil/)                     | Map with nil handling   |
| [list](./example/list/)                           | List injection          |
| [list-nil](./example/list-nil/)                   | List with nil handling  |
| [lazy](./example/lazy/)                           | Lazy injection          |
| [cycle](./example/cycle/)                         | Cycle detection example |
| [handler](./example/handler/)                     | Handler pattern         |
| [inject_method](./example/inject_method/)         | Method injection        |
| [test-return-error](./example/test-return-error/) | Error handling          |
| [http](./example/http/)                           | HTTP visualization      |

## 📖 Documentation

| Document                              | Description                              |
| ------------------------------------- | ---------------------------------------- |
| [Design Document](./docs/design.md)   | Architecture and detailed design         |
| [Audit Report](./docs/audit.md)       | Project audit, evaluation and comparison |
| [dixhttp README](./dixhttp/README.md) | HTTP visualization module documentation  |

## 📄 License

MIT
