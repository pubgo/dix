# dix

[![Go Reference](https://pkg.go.dev/badge/github.com/pubgo/dix/v2.svg)](https://pkg.go.dev/github.com/pubgo/dix/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/pubgo/dix)](https://goreportcard.com/report/github.com/pubgo/dix)

> **dix** is a lightweight yet powerful dependency injection framework for Go.

Inspired by [uber-go/dig](https://github.com/uber-go/dig), with support for advanced dependency management and namespace isolation.

[中文文档](./README_zh.md)

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

## 📖 Core API

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

### Startup Timeout / Slow Provider Warning

Control long-running providers during startup:

- Default provider timeout: `15s`
- Disable provider timeout explicitly: `dix.WithProviderTimeout(0)`
- Default slow provider warning threshold: `2s`
- Disable slow provider warning: `dix.WithSlowProviderThreshold(0)`

```go
di := dix.New(
    // Default `ProviderTimeout` is `15s`
    // Use `dix.WithProviderTimeout(0)` to disable provider timeout
    // Default `SlowProviderThreshold` is `2s`
    // Use `dix.WithSlowProviderThreshold(0)` to disable slow-provider warnings
    dix.WithProviderTimeout(2*time.Second),        // override default (default: 15s, 0 = disabled)
    dix.WithSlowProviderThreshold(300*time.Millisecond), // override default (default: 2s, 0 = disabled)
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
```

### dixhttp - Dependency Visualization 🆕

Web interface for visualizing dependency graph, **designed for large projects**:

```go
import (
    "github.com/pubgo/dix/v2/dixhttp"
    "github.com/pubgo/dix/v2/dixinternal"
)

server := dixhttp.NewServer((*dixinternal.Dix)(di))
server.ListenAndServe(":8080")
```

Visit `http://localhost:8080` to view dependency graph.

**Highlights**:
- 🔍 **Fuzzy Search** - Quickly locate types or functions
- 📦 **Package Grouping** - Collapsible sidebar browsing
- 🔄 **Bidirectional Tracking** - Show both dependencies and dependents
- 📏 **Depth Control** - Limit display levels (1-5 or all)
- 🎨 **Modern UI** - Tailwind CSS + Alpine.js

See [dixhttp/README.md](./dixhttp/README.md) for details.

## 🛠️ Development

```bash
# Run tests
task test

# Lint
task lint

# Build
task build
```

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
