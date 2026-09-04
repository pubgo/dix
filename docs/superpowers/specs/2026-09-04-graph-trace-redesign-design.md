# dix Graph / Trace / 调用链路重设计

> 状态:评审中 · 方案 A(图为中心)· 允许破坏性 API · trace 默认开 · 前后端一起重构
> 日期:2026-09-04 · 分期:P1 图与缓存 → P2 trace 容器化与调用树 → P3 埋点统一 → P4 大规模可视化与检索
> 规模目标:100+ provider、300+ 对象、几十个模块的容器下,展示/过滤/查询仍然可用(见 P4)

## 1. 背景与问题

现状是三套互不相通的系统:

| 子系统 | 数据模型 | 消费方 |
|---|---|---|
| 运行时 trace(`dixtrace`) | `Event`(stringly + attrs map),TraceID 为进程计数器 | 全局 MemorySink(5000 条 ring,常开)、`DIX_TRACE_FILE`、`/api/trace` |
| 步骤日志(`logDITrace`) | slog 文本 + diag JSONL record(fields map,kind: trace/error/llm) | `DIX_TRACE_DI`、`DIX_DIAG_FILE`、LLM 模式 |
| 依赖图 | ① `depGraph`(类型邻接 bool 集,仅环检测)② `extractDependencyData`(每请求反射现算) | ① `isCycle` ② dixhttp 前端 |

核心问题:① 依赖关系有三种互不相干的表示,埋点在 inject 链路上成对重复;② `depGraph` 无 provider/namespace/字段元数据且不对外;③ dixhttp 无缓存,每次请求全量反射;④ trace 与容器实例无关,多容器混流,TraceID 重启即撞;⑤ ring 5000 满即丢,启动期事件最先被冲掉;⑥ 调用链路需手工拼装 span 事件,无树视图;⑦ 查询为线性 substring 过滤,无聚合统计;⑧ #41 的 schema 冲突正是双模型的产物。

## 2. 设计原则

1. **单一事实来源**:容器持有运行时 `Graph`,声明依赖与实际解析都落在这里;环检测、dixhttp、trace 全部读它。
2. **调用链路 = 图的遍历轨迹**:一次 Inject 产出一棵 resolve tree,trace 事件是这棵树的序列化,不另造第二套链路数据。
3. **读路径无反射**:dixhttp 从 Graph 快照读,Provide 时失效、惰性重建。
4. **容器化观测**:Tracer 与 Graph 都挂在 Dix 上;全局 API 保留为兼容外壳。
5. **可见行为不变**:`DIX_*` 环境变量语义、现有锁测试、example 契约全部保持;破坏只发生在公开 API 签名与 dixhttp 响应的增字段上(已获准)。

## 3. P1 统一 Graph 模型(纯内部,行为不变)

### 3.1 数据结构(`dixinternal/graph.go`,新文件)

```go
type NodeID uint32

type NodeKind uint8 // NodeType | NodeProvider | NodeObject

type Node struct {
    ID       NodeID
    Kind     NodeKind
    Type     reflect.Type  // Type/Object: 依赖类型;Provider: 输出类型
    Group    string        // Object 节点的 namespace(默认 "default")
    Provider *providerFn   // Provider 节点回指(函数名/源码位置惰性取)
    Label    string        // 对外字符串视图,惰性生成后缓存
    Pkg      string
}

type EdgeKind uint8 // EdgeDeclared | EdgeProduced | EdgeResolved

type Edge struct {
    From, To  NodeID
    Kind      EdgeKind
    Field     string // 声明边:struct 输入的字段名
    Aggregate string // "" | "map" | "list"(聚合查询标记)
    Count     int64  // Resolved 边:累计解析次数(读写锁保护)
}

type Graph struct {
    mu        sync.RWMutex   // Dix 本身单线程,但 dixhttp 并发读,Graph 自带读锁
    nodes     []Node
    index     map[nodeKey]NodeID // (kind,type,group/providerFn) 去重
    edges     map[edgeKey]*Edge
    version   atomic.Uint64      // 每次 Provide +1,snapshot 失效判据
}
```

### 3.2 维护时机与索引

- **Provide**:成功注册后增量追加 `Provider` 节点 + `Produced` 边 + 每个输入的 `Declared` 边(含字段名/聚合标记)。O(输入数)。
- **Inject**:实际解析发生时,`getValue`/`executeProvider` 命中 `Resolved` 边并累加 Count(失败/超时也计数,带状态)。
- **对象缓存**:`processProviderOutput` 落缓存时追加/更新 `Object` 节点。
- **检索索引(P4 的地基,随 P1 一起建)**:Graph 维护三张内存倒排索引——按类型名分词、按 provider 函数名、按 module(pkg path);同时每个节点带运行时状态位(已实例化/未实例化、provider 有错、慢 provider)。这是"服务端过滤"的前提:前端不再拉全量自滤。

### 3.3 既有组件迁移

- **环检测**:`isCycle` 改为遍历 Graph 的 Declared 边;DFS 语义与 #57 锁定的确定性(字典序起点、trim 环)不变,`TestDetectCycleDeterministicOrder` 等锁测试原样通过。`depGraph`/`buildDependencyGraph`/`graphDirty` 删除。
- **dixhttp**:新增 `Graph.Snapshot(pkgFilter string, limit int)` 返回不可变数据(即现 `DependencyData` 结构,由 Graph 直接投影);`extractDependencyData` 删除;`version` 判脏 + 惰性重建 snapshot,同 version 重复请求零反射。
- **`RegisterGroupRules`**:语义不变(前端分组规则),仍为全局。

### 3.4 验收

- 现有全部测试绿(尤其 cycle 锁测试、dixhttp handler 契约测试、example 契约);
- 新增 Graph 单测:增量边正确性、snapshot 失效、并发读安全(`-race`)、Provide 后 version 递增;
- `/api/dependencies` 响应字段与现版本兼容(增字段允许)。

## 4. P2 trace 容器化与调用树

### 4.1 Tracer 容器化

- `Dix` 持有 `containerID string`(启动时随机 8 hex)与 `tracer *dixtrace.Tracer`(默认指向全局 default tracer,保持零配置)。
- `dixtrace.Event` 增加 `ContainerID string` 字段(只加不改,JSON 向后兼容);dixinternal 所有 Emit 点带上。
- **TraceID 随机化**:改为 `crypto/rand` 32 hex(对齐 OTel 语义);`Query`/`ParseQueryFromMap` 不变。
- 全局 `dixtrace.BeginSpan/Emit/QueryEvents` 保留,文档标注"默认容器外壳"。
- MemorySink 策略:默认仍开;`dix.New(WithTraceBuffer(n))` 可调(替代魔数 5000)。

### 4.2 调用树(TraceTree)

- MemorySink 增加**树索引**:按 `TraceID` 维护 `map[SpanID][]SpanID` 父子表(start 事件建,内存换查询)。
- 新 API:`dixtrace.QueryTree(traceID string, limit int) (TreeResult, error)`,`TreeResult` 为嵌套节点(引用 Event + children)。
- HTTP:`GET /api/trace-tree?trace_id=...`;`Inject` 的 root span operation 即 `inject`,树根即一次注入。
- 前端:依赖图页新增"调用链路"视图——选一次 Inject,渲染 resolve tree(节点带类型/ provider/耗时/状态)。

### 4.3 验收

- `trace_chain_test.go` 全部语义保持(嵌套规则不变);
- 新增:TreeResult 与事件流一致性、多容器事件隔离(按 containerID 过滤)、JSONL 旧文件可被新代码读取;
- example/context-inject 扩展断言树结构。

## 5. P3 埋点统一与 LLM 通道收敛

- Graph/resolve 过程发布结构化事件到容器内事件总线(就是 Tracer 的 sink 流,不新造机制)。
- **console(`di_trace ...`)与 diag file 降级为订阅者**:各实现一个 sink/适配器,格式与现状逐字节一致(锁测试 `TestDITraceLogsInInjectFlow`、`TestDiagFileConfiguredCollectsTraceErrorAndLLM` 原样通过)。
- `dix.go` 中 `logDITrace` 调用点全部删除,仅保留 span/事件发布一处;预期 dixinternal/dix.go 净减 200+ 行。
- **移除独立的 LLM 输出通道**(v2.1.0 后评审结论:同一错误的机器可读信息已存在三份,第四份纯属冗余):
  - 删 `DIX_LLM_DIAG_MODE` env 与 human/machine/dual 分支(machine-only 会整体丢弃人类日志,迫使"给人看"与"给 LLM 看"二选一,有害);
  - 删 `emitLLMDiagnosticLine` 与 stderr `DIX_LLM_DIAG {...}` 行;
  - diag 文件 record 从 `trace|error|llm` 三类收敛为 `trace|error` 两类(llm record 的全部字段 error record 均已包含);
  - 删 example/http 中重复解析该 env 的 `isMachineDiagMode`/`configureExampleLogOutput`;
  - **保留** `error_type` / `root_cause` / `hint` 字段体系(hint 文案是给 LLM 与人的真正价值),作为结构化字段走全部出口:stderr slog 行(现状已带 `dix.error_type=` 等属性)、diag JSONL error record、`/api/errors`;
  - **数据完整性是不变量**:任何出口(日志行/JSONL/API)必须携带完整结构化字段——"数据在,LLM 就能读",包括读人类可读日志;
  - LLM/agent 的消费契约定义为两条:终端 agent 读 stderr 的结构化 attrs;文件型 agent 读 diag JSONL 或 `/api/errors`、`/api/diagnostics`。不再有专门的"LLM 格式";
  - 兼容:`DIX_LLM_DIAG_MODE` 直接删除(不留 no-op 过渡;当前 LLM 完全可读人类可读日志,单独通道无存续价值)。

## 6. P4 大规模可视化与检索(100+ provider / 300+ 对象 / 几十模块)

核心判断:**大图不可渲染,只能分层与聚焦**。全量节点图在 500+ 节点时任何引擎都不可用,解法是让默认视图永远 small(节点数 <100),大范围检索交给服务端。

### 6.1 分层聚合(下钻式浏览)

Graph 的天然层次:`Module(pkg) → Type → Object`。API 支持粒度参数:

- `GET /api/graph?level=module` —— 默认视图:几十个模块节点,边为模块间依赖计数(完全可渲染);
- `GET /api/graph?level=type&module=xxx` —— 下钻单模块:该模块类型 + 跨模块依赖边界(接口边优先展示);
- `GET /api/graph?level=object&module=xxx&type=yyy` —— 再下钻:实例级(含 group/namespace、实例化状态)。

前端:点击模块 → 下钻;面包屑回退;跨模块边始终保留(这是"谁依赖谁"的关键信息)。

### 6.2 邻域子图(ego graph)——看大图的主交互

- `GET /api/graph?center=<type|provider>&depth=N&direction=both|deps|dependents`
- 以搜索命中的节点为中心,返回双向 N 跳子图(默认 depth=2)。大项目里"搜一个类型 → 看它的依赖闭包"是最高频动作,替代"全图截断"。
- 现有 depth control 语义升级为中心化邻域,不再是全图限制。

### 6.3 服务端检索

- `GET /api/search?q=xxx&kind=type|provider|module&module=...&state=instantiated|error|slow` —— 走 P1 倒排索引,毫秒级返回命中列表(带模块/状态标注),前端搜索框即搜即得;
- 运行时状态过滤:错误 provider、慢 provider、未实例化(惰性)对象一键筛出——诊断场景优先。

### 6.4 概览 dashboard

`/api/stats` 升级:模块数/类型数/provider 数/object 数 + resolved 次数 TopN、错误与慢 provider 列表、最近失败注入。作为落地页,先全局后下钻。

### 6.5 现网易用性问题清单(2026-09-04 实测 example/http,1440×900)

> 演示规模仅 2 包 / 23 provider / 45 依赖,问题已经全部显现;目标规模(100+/300+/几十模块)下不可用。

1. **单屏塞满、无信息分层**:落地即全量图,两行工具栏 12+ 控件(视图切换/布局/深度/聚合/调试匹配/前缀过滤/清除/分组清单/重置/下载 SVG/Mermaid/搜索)平铺无主次;没有"先总览后下钻"的入口。
2. **图不可读**:该规模下节点标签已小到不可辨;层级布局+深度截断只是部分缓解。
3. **诊断入口三处重复**:头部"Trace 诊断"、右下浮动"Trace 诊断入口"、诊断菜单里"调用链 Trace"指向同一功能;诊断能力(启动耗时/最近错误/错误类型)藏在下拉二级。
4. **图与调用链割裂**:Trace 是独立全屏面板,靠手输 trace_id 找链路,无法从图节点直达"该类型的解析树";裸计数器 trace_id(t-1/t-7)直接暴露给用户。
5. **右栏利用率低**:节点详情默认整屏留白;图例被右下浮动按钮遮挡。
6. **内部信息外泄**:页头暴露 "UI: trace-entry-v5-20260323" 版本标签。
7. **前端工程不可维护**:单文件 template.html 3877 行(内联 JS 2600+ 行);4 个 CDN 依赖(tailwind/alpine/vis-network/mermaid),内网即白屏;无组件化,迭代成本极高。

### 6.6 前端信息架构重写(五视图)

全局顶栏只保留:产品名、全局搜索(Cmd+K)、视图 Tab;现头部/浮动的重复入口全部合并。

| 视图 | 主任务 | 内容 |
|---|---|---|
| ① 概览(默认落地) | "系统现在怎么样" | 统计卡(模块/类型/provider/object/错误/慢)+ resolved TopN + 最近注入错误 + 模块级迷你图 |
| ② 依赖图 | 浏览依赖关系 | 默认模块级聚合图 → 下钻类型级 → 对象级;工具栏只留粒度/模块/聚焦;点击节点开抽屉(详情 + 运行时状态 + deps/dependents N 跳邻域 + "查看解析链路"直达调用树) |
| ③ 检索 | 找到目标 | 全局搜索 + 状态过滤(错误/慢/未实例化);选中即跳对应视图并聚焦 |
| ④ 调用链 | 看一次注入的树 | Trace 列表(错误优先/时间倒序,trace_id 用随机 ID 但对用户展示为"时间+首个操作")+ 树视图;支持从②的节点详情跳入并预过滤 |
| ⑤ 诊断 | 排查启动问题 | 现诊断菜单四项(启动耗时/最近错误/错误类型说明)升为独立视图、表格化 |

交互原则:一个视图一个主任务;工具栏按视图局部化;高频(搜索/下钻)与低频(导出/重置)分离,低频收进"更多"。

### 6.7 前端工程改造

- **拆文件**:go:embed 从单 template.html 改为 embed 静态目录(dist/),按视图拆分 JS/CSS;CDN 依赖本地化打包(内网可用是硬要求);
- **构建方式**:P4 开工时 spike 二选一——(a)保持无构建(多文件 + 原生 ESM + Alpine),(b)引入 Vite 产物 + go:embed dist(组件化更顺);以"内网可用、可维护、包体"三标准决策;
- **数据层**:前端只消费 P1/P2 新 API(snapshot/分页/搜索/trace-tree),不再拉全量、不在前端做过滤聚合。

### 6.8 渲染引擎选型

- 分层 + 邻域策略下默认视图节点数 <100,现有 vis-network 可继续承担;
- P4 开工时做一次 **渲染 spike**:go-echarts(graph 系列)/ sigma.js(WebGL)/ 保持 vis-network,用 100 模块真实规模数据对比帧率与交互,选型结果并入 #20 的结论;
- 备选视图:**模块依赖矩阵**(module × module 热力图),几十模块规模下比节点图更易读,作为补充视图评估。

## 7. 兼容与破坏性变更清单

| 项 | 变更 | 缓解 |
|---|---|---|
| `dixinternal` 内部符号 | `depGraph`/`buildDependencyGraph`/`extractDependencyData` 删除 | 内部包,无外部影响 |
| `dixtrace.Event` | 只增字段(ContainerID) | JSON 向后兼容 |
| `dixtrace` TraceID 格式 | 计数器 → 随机 32hex | 已获准破坏;无格式依赖的公开承诺 |
| `dixhttp` 响应 | 增字段(container_id、节点 id、调用树端点) | 向后兼容 |
| `/api/graph` 语义 | level 参数成默认;全量 object 列表改为分页/下钻 | 前端同期重构(已获准) |
| dixhttp 前端 | 五视图信息架构重写,template.html 拆分与本地化 | 已获准"前后端一起" |
| `DIX_LLM_DIAG_MODE` / `DIX_LLM_DIAG` 行 / diag `kind:llm` | P3 直接移除(无过渡期),error 字段体系保留 | `/api/diagnostics` 的 kind 枚举文档同步 |
| `dix.New` | 新 Option(WithTraceBuffer) | 纯增量 |

## 8. 分期交付

| 期 | PR 内容 | 出口标准 |
|---|---|---|
| P1 | graph.go(含检索索引)+ depGraph/extractDependencyData 迁移 + snapshot 缓存 + Graph 单测 | 全部现有测试原样绿;dixhttp 重复请求零反射(benchmark 佐证) |
| P2 | tracer 容器化 + 随机 TraceID + TraceTree API + 前端调用树视图 | trace_chain 语义保持;多容器隔离测试;前后端联调通过 |
| P3 | 埋点订阅化 + dix.go 瘦身 | console/diag 输出逐字节不变(锁测试保证);dix.go 行数下降 |
| P4 | 分层聚合 API + 邻域子图 + 服务端检索 + 五视图前端重写(含工程拆分/CDN 本地化)+ dashboard + 渲染 spike 选型(结论并入 #20) | 100 provider/300 object/40 模块规模的演示容器:默认视图节点数 <100;搜索→邻域子图全程无卡顿;全量 JSON 不再一次性下发;内网无 CDN 可完整使用 |

每期独立发 PR、独立可回滚;每期同步 `docs/design.md`(双语)与 changelog。
