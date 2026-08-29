# Dix Dependency Injection Framework - Design Document

[中文文档](./design_zh.md)

> Last updated: 2026-08-29, against `v2` (dixinternal as of v2.0.1+).

## 1. Overview

`dix` is a lightweight dependency injection (DI) framework for Go, implemented using reflection. Inspired by `dig`, it aims to decouple component dependencies through automated dependency resolution and lifecycle management.

The core of `dix` manages dependency injection for addressable types, mainly including `func`, `ptr` (pointer), and `interface`.

Core features include:
*   **Constructor Injection**: Register constructors via `Provide`.
*   **Struct/Field Injection**: Fill struct fields via `Inject`.
*   **Advanced Type Support**: Support for `Interface`, `Map`, `Slice` with automatic aggregation.
*   **Cycle Detection**: Built-in graph algorithm with a cached dependency graph.
*   **Web Visualization**: HTTP module for interactive dependency graph visualization.
*   **Method Injection**: Support setter injection via `DixInject` prefix methods.
*   **Namespace Grouping**: Support grouping dependencies by namespace.
*   **Safe APIs**: `TryProvide`/`TryInject` variants that return errors instead of panicking.
*   **Runtime Diagnostics**: Provider execution stats, recent error records, structured tracing (`dixtrace`), and a JSONL diagnostics file.

## 2. Core Architecture

The core of `dix` is a container struct named `Dix`, which maintains a registry of providers and a cache of instantiated objects.

### 2.1 Supported Types

`dix` strictly limits the types that can be managed as dependencies. The core principle is **only addressable or reference types are supported**. This design ensures stability of dependency relationships and controllability of object lifecycles.

*   **Pointer**:
    *   The most common type, usually pointing to a struct instance (e.g., `*Service`).
    *   Pointers guarantee singleton behavior throughout the application lifecycle with shared state.
*   **Interface**:
    *   Supports binding concrete implementations to interface definitions (e.g., `Service` interface implemented by `*ServiceImpl`).
    *   Key to implementing the Dependency Inversion Principle (DIP).
*   **Func**:
    *   Functions are first-class citizens and can be injected as dependencies.
    *   Commonly used for factory functions, middleware, or callbacks.

*Note: Basic types (e.g., `int`, `string`, `bool`) cannot be directly injected as dependencies; they must be wrapped in the above types (usually as struct fields). Providers declaring unsupported input/output types are rejected at registration time.*

### 2.2 Data Structures

```go
type Dix struct {
    // Configuration options
    option Options

    // Provider registry
    // key: output type (reflect.Type)
    // value: list of provider functions for this type
    providers map[outputType][]*providerFn

    // Object cache pool (singleton pattern)
    // key: output type -> group -> value list
    objects map[outputType]map[group][]value

    // Initialization state flags to prevent duplicate Provider execution
    initializer map[reflect.Value]bool

    // Providers whose call timed out; they are never re-executed
    timedOut map[reflect.Value]bool

    // Cached dependency graph for cycle detection,
    // rebuilt lazily after providers change
    depGraph   map[reflect.Type]map[reflect.Type]bool
    graphDirty bool

    // Runtime stats per provider (call count, durations, last error)
    providerStats map[reflect.Value]*providerRuntimeStat

    // Ring buffer of recent inject/provider failures for diagnostics
    recentErrors []recentErrorRecord
}
```

### 2.3 Core Flow

1.  **Register (Provide)**: User registers constructor -> Parse function signature (input/output) -> Validate types (unsupported input/output types fail here, not at Inject time) -> Store in `providers` table -> Mark the dependency graph stale.
2.  **Invoke (Inject)**: User requests object injection -> Cycle check on the cached graph -> Recursively find dependencies -> Execute Provider function (with timeout control) -> Cache result -> Fill target.

## 3. Detailed Design

### 3.1 Provider Registration

The `Provide` method registers constructors into the container. `Provide` panics on invalid registrations; `TryProvide` returns an error instead.

#### 3.1.1 Input Parameter Types
Provider function parameters declare the component's dependencies:

*   **Func / Ptr / Interface**:
    *   **Behavior**: Declared as direct dependency.
    *   **Resolution**: Container finds matching instance from registered Providers.
*   **Struct**:
    *   **Behavior**: **Recursive Dependency Injection**.
    *   **Resolution**: Framework recursively traverses all **exported fields**. If a field type is supported (Func/Ptr/Interface/aggregate types), the container automatically finds and injects the corresponding instance.
*   **Slice (`[]T`)**:
    *   **Behavior**: Declared as aggregate dependency (List).
    *   **Resolution**: Container finds all Providers that can provide type `T` and aggregates results from the **default group** into a slice.
*   **Map (`map[string]T`)**:
    *   **Behavior**: Declared as aggregate dependency (Map).
    *   **Resolution**: Container finds all Providers for type `T`, using their returned Keys (default "default") to form a Map.

*Parameters that cannot be parsed into any injectable input (e.g. `func(i int)`) are rejected at registration time — registering a provider whose signature cannot be satisfied would otherwise panic at Inject time with an arity mismatch.*

#### 3.1.2 Return Value Types
Provider function return values define what components it provides to the container.

**Constraints**:
Provider function must return **1 or 2** values.
*   1 value: That value is the provided component.
*   2 values: Second value must be `error` type. If `error` is not nil, container stops initialization and reports error.

Supported types:

*   **Func / Ptr / Interface**:
    *   **Behavior**: Register as standard Provider for that type.
*   **Slice (`[]T`)**:
    *   **Behavior**: Register as list Provider for type `T`.
    *   **Effect**: Allows one Provider to provide multiple `T` instances at once.
*   **Map (`map[string]T`)**:
    *   **Behavior**: Register as map Provider for type `T`.
    *   **Effect**: Allows Provider to specify component Key.
*   **Struct**:
    *   **Behavior**: **Recursive Auto-Flattening**.
    *   **Effect**: Framework recursively traverses all **exported fields**. Supported field types are registered as separate Providers.

*Unsupported output kinds (e.g. `func() int`) fail registration instead of being silently ignored.*

### 3.2 Dependency Resolution & Injection

Injection is driven by `Inject` or internal `getValue`, using **Lazy Evaluation** strategy.

#### 3.2.1 Injection Target Types

`Inject` function supports injection into struct pointers or functions.

**1. Struct Pointer**
When passing `&MyStruct{}`, framework scans fields for injection:
*   **Func / Ptr / Interface**: Find and inject corresponding type instance.
*   **Struct**: **Recursive Injection** - framework continues injecting into nested struct fields.
*   **Slice / Map**: Execute aggregation injection logic.
*   **Method Injection**: Auto-scan and execute methods prefixed with `DixInject` (Setter injection variant).

**2. Function**
When passing a function (e.g., `dix.Inject(func(a *A, b *B){ ... })`):
*   Parameters are treated as dependencies.
*   Resolution logic same as Provider input parameters (see 3.1.1).
*   Commonly used for initialization or startup hooks.

#### 3.2.2 Internal Storage & Resolution Strategy

When multiple Providers exist for the same type, `dix` uses grouping (Group/Key) mechanism. The underlying storage is `map[group][]value`, where `group` is a label (default "default").

Resolution strategies for different dependency declarations:

1.  **Single Value (`T`)**: Query **default group**, take **last value**.
2.  **List (`[]T`)**: Query **default group**, take **all values**.
3.  **Map (`map[string]T`)**: Query **all groups**, take **last value** per group.
4.  **Full Map (`map[string][]T`)**: Query **all groups**, take **all values** per group.

### 3.3 Options

Container options are set via `New(opts...)`; every `Inject`/`TryInject` call may pass additional options, and **caller-level options take precedence over container defaults**.

| Option | Default | Semantics |
| :--- | :--- | :--- |
| `AllowValuesNull` / `WithValuesNull()` | `true` | Tolerates missing **map/list** dependencies by resolving them as empty collections. Missing single-value dependencies always fail. |
| `WithRejectEmptyCollections()` | — | Rejects missing map/list dependencies instead of resolving them as empty collections. |
| `ProviderTimeout` / `WithProviderTimeout(d)` | `15s` | Max execution time of one provider call; `0` disables timeout. A timed-out call cannot be aborted: the provider is marked as failed and is **never re-executed** by later injections. |
| `SlowProviderThreshold` / `WithSlowProviderThreshold(d)` | `2s` | Warn when a provider call exceeds the threshold; `0` disables the warning. |

### 3.4 Cycle Detection

To prevent infinite recursion, `dix` checks a dependency graph before injection.

*   **Algorithm**: Depth-First Search (DFS).
*   **Implementation**: `detectCycle` / `isCycle` in `dixinternal/cycle-check.go` and `dixinternal/util.go`.
*   **Logic**: Build `map[reflect.Type]map[reflect.Type]bool` adjacency list, traverse to find back edges. If a cycle is found, the reported path is **trimmed to the repeated node** (e.g. `X -> A -> B -> A` is reported as `A -> B -> A`) so the message does not include cycle-external nodes.
*   **Caching**: The graph is cached on the container and rebuilt lazily only after `Provide` marks it stale — repeated injections do not re-build the graph.

### 3.5 Error Handling & Runtime Diagnostics

Errors are logged with rich context (provider name, output/input types, stage, root cause, hint) and recorded in-memory:

*   **Panic Recovery**: Uses `defer recover` when executing user code (Provider functions) to prevent application crashes.
*   **Safe APIs**: `TryProvide` and `TryInject` methods return errors instead of panicking, suitable for scenarios where graceful error handling is preferred.
*   **Recent Errors**: The container keeps a bounded ring buffer of recent failures (`GetRecentErrors`), exposed via `/api/errors`.
*   **Provider Stats**: Per-provider call count and durations (`GetProviderRuntimeStats`), exposed via `/api/runtime-stats`; slow providers emit warnings.
*   **Diagnostics File**: When `DIX_DIAG_FILE` is set, structured error/LLM records are appended as JSONL for post-mortem analysis (`ReadDiagFileRecords`).

### 3.6 Tracing

`dixtrace` provides span/trace instrumentation of the resolve/provider call chain (`BeginSpanCtx`, `Emit`):

*   In-memory ring buffer queryable via `/api/trace`, plus JSONL export.
*   File export is enabled **only** by setting `DIX_TRACE_FILE` explicitly. There is no fallback to `DIX_DIAG_FILE` — the two files use different JSON schemas by design.

### 3.7 Visualization

`dix` provides HTTP-based visualization through the `dixhttp` module (`NewServer(di *dix.Dix)`):

*   **Web Interface**: Modern UI built with Tailwind CSS + Alpine.js + vis-network
*   **Features**:
    *   Fuzzy search for types and functions
    *   Package-based grouping with collapsible sidebar
    *   Bidirectional dependency tracking (upstream & downstream)
    *   Depth control (1-5 levels or all)
    *   Interactive graph with drag, zoom, and click
*   **RESTful API**: JSON endpoints for programmatic access
    *   `/api/stats` - Summary statistics
    *   `/api/runtime-stats` - Provider runtime metrics
    *   `/api/errors` - Recent failure records
    *   `/api/diagnostics` - Diag-file record query
    *   `/api/trace` - Trace event query
    *   `/api/packages` - Package list
    *   `/api/package/{name}` - Package details
    *   `/api/dependencies` - Full dependency data
    *   `/api/type/{name}` - Type-specific dependency chain
    *   `/api/group-rules` - Group aggregation rules

### 3.8 Extension Modules

#### 3.8.1 Global Container
`dixglobal` package provides a global container instance for sharing across application parts.

#### 3.8.2 Context Support
`dixcontext` package provides functionality to store container instance in context for passing through request chains.

#### 3.8.3 HTTP Visualization
`dixhttp` package provides HTTP server for visualizing dependency graphs with interactive web interface. `NewServer`/`NewServerWithOptions` accept `*dix.Dix` directly. See [dixhttp/README.md](../dixhttp/README.md) for details.

#### 3.8.4 Tracing
`dixtrace` package provides span-based tracing of the injection and provider call chains. See 3.6.

## 4. Module Breakdown

| File | Responsibility |
| :--- | :--- |
| `dix.go` | Public API wrapper with generics support (`Inject[T]`, `InjectT[T]`, `Provide`). |
| `dixinternal/api.go` | Core public API (`New`, `Provide`, `TryProvide`, `Inject`, `TryInject`). |
| `dixinternal/dix.go` | Core logic: `newDix` initialization, `inject` recursive flow, Provider registration. |
| `dixinternal/provider.go` | `providerFn` struct definition, encapsulates reflection call details and timeout control. |
| `dixinternal/option.go` | Configuration options (`WithValuesNull`, `WithRejectEmptyCollections`, `WithProviderTimeout`, `WithSlowProviderThreshold`). |
| `dixinternal/cycle-check.go` | Circular dependency detection over the cached dependency graph. |
| `dixinternal/util.go` | Utility functions: reflection Map/Slice creation, output type handling, graph building, cycle DFS. |
| `dixinternal/diag_file.go` | JSONL diagnostics file writer/reader (`DIX_DIAG_FILE`). |
| `dixinternal/logger.go` | Logging system integration with structured output. |
| `dixtrace/` | Span/trace instrumentation, in-memory sink, optional JSONL file sink. |
| `dixglobal/` | Global container module with singleton instance. |
| `dixcontext/` | Context integration module for storing container in context. |
| `dixhttp/` | HTTP visualization module with modern web interface. |
| `example/` | Independent Go module with runnable examples (compiled in CI). |

## 5. Thread Safety

**Important**: The `Dix` container is **NOT thread-safe** by design. Users should not call `Provide`/`Inject` concurrently on the same container instance. This is a deliberate design choice for performance, as dependency injection typically happens during application initialization (single-threaded).

For concurrent scenarios, consider:
1. Complete all `Provide` calls before starting concurrent operations
2. Use separate container instances per goroutine
3. Wrap container access with external synchronization if needed

## 6. Summary

`dix` is a feature-complete Go dependency injection container. It trades some runtime performance (through reflection) for great development flexibility - acceptable during initialization phase.

Design highlights:
*   Full utilization of Go language features (multi-return for error handling, generics-friendly wrappers)
*   Elegant support for collection type injection
*   Fast-failing registration: invalid provider signatures are rejected at Provide time
*   Cached cycle detection with precise, trimmed cycle paths
*   Built-in runtime diagnostics: provider stats, recent errors, tracing, diagnostics file
*   Rich extension ecosystem (global container, context integration, web visualization)
*   Safe API variants for graceful error handling

This makes `dix` a complete dependency injection solution for Go applications.
