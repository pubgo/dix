# Dix HTTP Visualization Module

This module provides an HTTP server to visualize dependency relationships in the Dix dependency injection container. **Designed for large projects**, with support for fuzzy search, package grouping, bidirectional dependency tracking, and depth control.

[中文文档](./README_zh.md)

## Features

- 📊 **Interactive Visualization** - Modern UI built with vis.js + Tailwind CSS + Alpine.js
- 🔍 **Global Fuzzy Search** - Quickly search for type names or function names to view dependencies
- 📦 **Package Grouping** - Collapsible left panel to browse by package
- 🔄 **Bidirectional Dependency Tracking** - Show both upstream (dependencies) and downstream (dependents)
- 📏 **Depth Control** - Limit dependency graph display levels (1-5 or all)
- 🎨 **Multiple Layouts** - Support hierarchical and force-directed layouts
- 🧩 **Group Rules (Prefix Aggregation)** - Aggregate nodes by package/prefix rules
- 🔎 **Prefix Filter** - Show only nodes/providers matching a prefix
- 🧭 **Group Subgraph** - View a group's internal + upstream/downstream dependencies
- ⏱️ **Startup Runtime Stats** - Show all providers' startup durations (total/avg/last, call count), with executed-only filter (`call_count > 0`)
- 🗂 **Diagnostic File Query** - When `DIX_DIAG_FILE` is set, UI can query and display `trace/error/llm` JSONL records for troubleshooting
- 🧵 **Trace Timeline Query** - Query unified in-memory trace events from `dixtrace` via `/api/trace` (with rich filters); for file persistence, `DIX_TRACE_FILE` is preferred, and falls back to `DIX_DIAG_FILE` when unset
- 📡 **RESTful API** - Provide JSON format dependency data
- 🧩 **Mermaid Export/Preview** - Generate Mermaid flowcharts for current graph (respects grouping/filtering)

## Quick Start

```go
package main

import (
    "log"
    "github.com/pubgo/dix/v2"
    "github.com/pubgo/dix/v2/dixhttp"
    "github.com/pubgo/dix/v2/dixinternal"
)

func main() {
    // Create Dix container
    di := dix.New()
    
    // Register providers
    dix.Provide(di, func() *Config {
        return &Config{}
    })
    
    dix.Provide(di, func(c *Config) *Database {
        return &Database{Config: c}
    })
    
    dix.Provide(di, func(db *Database) *UserService {
        return &UserService{DB: db}
    })
    
    // Create and start HTTP server
    server := dixhttp.NewServer((*dixinternal.Dix)(di))
    log.Println("Server starting at http://localhost:8080")
    if err := server.ListenAndServe(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

Open browser and visit `http://localhost:8080` to view the dependency graph.

### Base Path / Prefix

If you need to mount the UI and API under a path prefix (e.g. behind a gateway), use `WithBasePath`:

```go
server := dixhttp.NewServerWithOptions(
  (*dixinternal.Dix)(di),
  dixhttp.WithBasePath("/dix"),
)
// Visit http://localhost:8080/dix/
```

### Reverse Proxy Auth Example (Recommended)

`dixhttp` itself does not include auth middleware. For production, place it behind a reverse proxy
with authentication and network restrictions.

Example (Nginx + Basic Auth):

```nginx
location /dix/ {
    auth_basic "Restricted Dix";
    auth_basic_user_file /etc/nginx/.htpasswd;

    proxy_pass http://127.0.0.1:8080/dix/;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

Practical security checklist:

- Bind the dixhttp server to loopback/internal interface only.
- Keep access behind auth (Basic Auth / SSO / gateway token).
- Restrict by source IP/CIDR where possible.
- Avoid exposing diagnosis endpoints (`/api/errors`, `/api/diagnostics`, `/api/trace`) to public internet.

## UI Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  🔗 Dix Dependency Visualization            📦 8 Pkgs  ⚡ 42 Provs  │
├──────────┬────────────────────────────────────────────┬─────────────┤
│ 📦 Pkgs  │  [Providers] [Types]  Layout: Hier  Depth: 2  🔍 Search  │ 📋 Details  │
│          │                                            │             │
│ 📁 All   │                                            │  Function   │
│  42      │           ┌─────────┐                      │  xxx.func   │
│          │           │ Config  │                      │             │
│ .../app  │           └────┬────┘                      │  Output     │
│  12      │                │                           │  *Config    │
│          │           ┌────▼────┐                      │             │
│ .../db   │           │Database │                      │  Inputs     │
│  8       │           └────┬────┘                      │  (clickable)│
│          │                │                           │             │
│          │        ┌───────▼───────┐                   │ [View Deps] │
│          │        │  UserService  │                   │             │
│          │        └───────────────┘                   │             │
├──────────┴────────────────────────────────────────────┼─────────────┤
│  « Collapse                                           │   Legend    │
└───────────────────────────────────────────────────────┴─────────────┘
```

### Three-Panel Layout

| Area                          | Description                                               |
| ----------------------------- | --------------------------------------------------------- |
| **Left - Package List**       | Provider list grouped by package, searchable, collapsible |
| **Center - Dependency Graph** | Interactive graph with drag, zoom, click support          |
| **Right - Details Panel**     | Show selected node details with clickable navigation      |

## Core Features

### 🔍 Global Fuzzy Search

The search box on the right side of the toolbar supports:

- **Fuzzy Matching** - Enter keywords to match type names or function names
- **Real-time Suggestions** - Show matching results dropdown (max 20)
- **Quick Jump** - Click result or press Enter to view dependency graph
- **Category Labels** - Results marked as `Provider` or `Type`

### 🔄 Bidirectional Dependency Tracking

After searching or clicking a type, the system shows that type as center:

```
     ┌──────────┐
     │ Upstream │  ← Green nodes
     │ (Config) │
     └────┬─────┘
          │
          ▼
     ┌──────────┐
     │  Target  │  ← Yellow highlight
     │(Database)│
     └────┬─────┘
          │
          ▼
     ┌──────────┐
     │Downstream│  ← Red nodes
     │(Service) │
     └──────────┘
```

**Color Legend**:
- 🟡 **Yellow** - Target node (searched object)
- 🟢 **Green** - Dependencies (upstream, what target depends on)
- 🔴 **Red** - Dependents (downstream, what depends on target)

### 📏 Depth Control

Depth determines how many levels to expand up/down:

| Depth | Description                         | Use Case                            |
| ----- | ----------------------------------- | ----------------------------------- |
| 1     | Only direct dependencies/dependents | Quick view of direct relationships  |
| 2     | Two levels (default)                | Recommended for daily use           |
| 3-5   | More levels                         | Track complex dependency chains     |
| All   | Show complete dependency tree       | Small projects or specific analysis |

**Example**: Assume dependency chain is `Config → Database → UserService → Handler`

Search `UserService` with different depths:
- Depth 1: `Database ← UserService → Handler`
- Depth 2: `Config ← Database ← UserService → Handler`

### 🧩 Group Rules (Prefix Aggregation)

You can aggregate nodes by **package or prefix** using group rules. Rules can be configured:

- **In UI** (group list)
- **From backend** via `RegisterGroupRules` (recommended for production)

Backend registration:

```go
import "github.com/pubgo/dix/v2/dixhttp"

dixhttp.RegisterGroupRules(
  dixhttp.GroupRule{
    Name: "service",
    Prefixes: []string{
      "github.com/acme/app/service",
      "github.com/acme/app/internal/service",
    },
  },
  dixhttp.GroupRule{
    Name: "router",
    Prefixes: []string{"github.com/acme/app/router"},
  },
)
```

The UI will auto-load `/api/group-rules` if local rules are empty.

### 🔎 Prefix Filter

The toolbar provides a **Prefix Filter** field. It filters the current graph to show only nodes/providers
whose package/type/function name contains the given prefix. This works in:

- Providers/Types view
- Type-focused dependency view
- Group subgraph view

### 🧭 Group Subgraph

Click a virtual group node to open the **group detail panel**, then click **View group graph** to see:

- Internal nodes
- Upstream & downstream dependencies
- Depth control applied from the toolbar

### 📦 Package Grouping

Left panel features:
- **Package List** - Show all packages with Provider counts
- **Search Filter** - Quickly locate specific packages
- **Click Filter** - Show only that package's dependencies
- **Collapse** - Click `«` button to collapse sidebar, expand graph area

## Interactions

| Operation                 | Effect                                      |
| ------------------------- | ------------------------------------------- |
| **Single Click**          | Show details in right panel                 |
| **Double Click**          | Show dependency graph centered on that node |
| **Drag Node**             | Move node position                          |
| **Scroll Zoom**           | Zoom in/out graph                           |
| **Click Type in Details** | Jump to view that type's dependencies       |

## Mermaid Support

The toolbar includes a **Mermaid** button. It generates a Mermaid `flowchart` from the **current graph view** (including grouping, depth, and prefix filters), opens a preview modal, and lets you copy the Mermaid source.

**Typical usage**:
1. Adjust view / grouping / filters.
2. Click **Mermaid**.
3. Copy the generated Mermaid text or use the preview.

## API Endpoints

### GET `/`
Returns HTML visualization page

### GET `/api/stats`
Returns summary statistics

```json
{
  "provider_count": 42,
  "object_count": 15,
  "package_count": 8,
  "edge_count": 67
}
```

### GET `/api/runtime-stats?limit=20`
Returns provider runtime metrics sorted by total duration (desc), useful for finding slow startup components.

```json
[
  {
    "function_name": "main.NewUserService",
    "output_type": "*service.UserService",
    "call_count": 1,
    "total_duration": 3456789,
    "average_duration": 3456789,
    "last_duration": 3456789,
    "last_run_at_unix_nano": 1700000000000000000
  }
]
```

### GET `/api/errors?limit=50`
Returns recent `Inject` / `TryInject` errors (latest first), useful when startup injection fails before full initialization.

```json
[
  {
    "operation": "provider_execute",
    "component": "main.main.func12",
    "stage": "resolve_input",
    "provider_function": "main.main.func12",
    "output_type": "*main.UserService",
    "input_type": "*main.Database",
    "root_cause": "value not found: type=*main.Database ...",
    "message": "failed to get input value for provider: value not found: type=*main.Database ...",
    "occurred_at_unix_nano": 1700000000000000000
  }
]
```

### GET `/api/diagnostics?kind=trace&q=provider&event=provider.call.start&limit=200`
Reads and filters JSONL records from `DIX_DIAG_FILE`.

If `DIX_DIAG_FILE` is not set, response returns `enabled=false` and empty records.

```json
{
  "enabled": true,
  "path": "/tmp/dix-diag.jsonl",
  "exists": true,
  "total": 42,
  "returned": 42,
  "next_before_id": 0,
  "records": [
    {
      "record_id": 128,
      "source": "dix",
      "pid": 12345,
      "process": "my-app",
      "hostname": "dev-mac",
      "trace_di": true,
      "llm_diag_mode": "dual",
      "kind": "trace",
      "event": "provider.call.start",
      "occurred_at_unix_nano": 1700000000000000000,
      "fields": {
        "provider": "github.com/acme/app.main.NewDB"
      }
    }
  ]
}
```

### GET `/api/trace?operation=provider&status=error&limit=200`
Returns in-memory unified trace events from `dixtrace`.

File sink behavior:

- Prefer `DIX_TRACE_FILE` when configured.
- If `DIX_TRACE_FILE` is unset and `DIX_DIAG_FILE` is set, trace file sink reuses `DIX_DIAG_FILE` in append mode (single-file troubleshooting setup).

Supported filters:

- `trace_id`, `operation`, `status`, `event`, `component`, `provider`, `output_type`, `q`
- `limit`, `before_id`, `since_unix_nano`, `until_unix_nano`

```json
{
  "enabled": true,
  "total": 2,
  "returned": 2,
  "records": [
    {
      "id": 102,
      "operation": "provider",
      "phase": "call.failed",
      "event": "provider.call.failed",
      "status": "error",
      "provider_function": "github.com/acme/app.main.NewDB",
      "output_type": "*db.Client",
      "error": "dial tcp timeout",
      "timed_out": true,
      "occurred_at_unix_nano": 1700000000000000000
    }
  ]
}
```

### GET `/api/packages`
Returns package list

```json
[
  {
    "name": "github.com/example/app/service",
    "provider_count": 12,
    "types": ["*service.UserService", "*service.OrderService"]
  }
]
```

### GET `/api/dependencies?package=xxx&limit=100`
Returns dependency data, supports package filtering

```json
{
  "providers": [
    {
      "id": "provider_*main.ServiceA_0",
      "output_type": "*main.ServiceA",
      "output_pkg": "github.com/example/app/service",
      "function_name": "main.NewServiceA",
      "function_pkg": "github.com/example/app",
      "input_types": ["*main.Config"],
      "input_pkgs": ["github.com/example/app/config"]
    }
  ],
  "objects": [...],
  "edges": [...]
}
```

### GET `/api/package/{packageName}`
Returns Provider details for specified package

### GET `/api/type/{typeName}?depth=2`
Returns dependency chain for specified type

```json
{
  "root_type": "*service.UserService",
  "depth": 2,
  "nodes": [
    {"id": "*service.UserService", "type": "*service.UserService", "package": "...", "level": 0}
  ],
  "edges": [
    {"from": "*db.Database", "to": "*service.UserService", "type": "dependency"}
  ]
}

### GET `/api/group-rules`
Returns backend-registered group rules (used as UI defaults)

```json
[
  {"name": "service", "prefixes": ["github.com/acme/app/service"]}
]
```
```

## Tech Stack

- **Backend**: Go standard library `net/http`
- **Frontend**: 
  - [Tailwind CSS](https://tailwindcss.com/) - Styling
  - [Alpine.js](https://alpinejs.dev/) - Reactive interactions
  - [vis-network](https://visjs.github.io/vis-network/) - Graph rendering
- **Template**: Go embed single-file HTML

## Use Cases

✅ **Recommended**:
- Large projects (100+ providers)
- Modular architecture needing package-based viewing
- Tracking specific type dependency chains
- Debugging circular dependency issues
- Onboarding new team members

⚠️ **Notes**:
- Production environment should restrict access (internal network or dev only)
- Very large projects (1000+ providers) should use depth limits

## Example

See `example/http/main.go` for complete example.
