package dixinternal

import (
	"reflect"
	"sort"
	"strings"
)

// graph_query.go 提供面向可视化/检索的容器级查询:
// 检索(SearchNodes)、模块聚合(ModuleGraph)、邻域子图(EgoGraph)、
// 解析热度(ResolvedTopN)与问题 provider(ProblemProviders)。
// 全部为只读投影,需同时读 Graph 与 providerStats,故放在 Dix 上而非 Graph 上。

// SearchHit 是检索命中的节点摘要。
type SearchHit struct {
	ID       uint32 `json:"id"`
	Kind     string `json:"kind"` // type|provider|object
	Label    string `json:"label"`
	Pkg      string `json:"pkg,omitempty"`
	Group    string `json:"group,omitempty"`
	State    string `json:"state,omitempty"` // instantiated|error|slow
	Provider string `json:"provider,omitempty"`
}

func nodeKindName(k NodeKind) string {
	switch k {
	case NodeType:
		return "type"
	case NodeProvider:
		return "provider"
	case NodeObject:
		return "object"
	}
	return "unknown"
}

// SearchNodes 按关键字/类别/模块前缀/运行时状态过滤图节点。
func (dix *Dix) SearchNodes(q, kind, module, state string, limit int) []SearchHit {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	qLower := strings.ToLower(strings.TrimSpace(q))
	kind = strings.ToLower(strings.TrimSpace(kind))
	state = strings.ToLower(strings.TrimSpace(state))
	module = strings.TrimSpace(module)

	g := dix.graph
	g.mu.RLock()
	defer g.mu.RUnlock()

	// 已实例化类型集合:存在 Object 节点 (type, group) 即视为实例化
	instantiated := make(map[reflect.Type]bool, len(g.nIndex))
	for k := range g.nIndex {
		if k.kind == NodeObject {
			instantiated[k.typ] = true
		}
	}

	hits := make([]SearchHit, 0, 16)
	for _, n := range g.nodes {
		kindName := nodeKindName(n.Kind)
		if kind != "" && kindName != kind {
			continue
		}
		if module != "" && !strings.HasPrefix(n.Pkg, module) {
			continue
		}
		if qLower != "" {
			hay := strings.ToLower(n.Label)
			if n.Provider != nil {
				hay += " " + strings.ToLower(GetFnName(n.Provider.fn))
			}
			if !strings.Contains(hay, qLower) {
				continue
			}
		}

		hit := SearchHit{ID: uint32(n.ID), Kind: kindName, Label: n.Label, Pkg: n.Pkg, Group: n.Group}
		if n.Provider != nil {
			hit.Provider = GetFnName(n.Provider.fn)
		}
		switch {
		case n.Kind == NodeType && instantiated[n.Type]:
			hit.State = "instantiated"
		case n.Kind == NodeProvider:
			hit.State = dix.providerState(n.Provider)
		}
		if state != "" && hit.State != state {
			continue
		}

		hits = append(hits, hit)
		if len(hits) >= limit {
			break
		}
	}
	return hits
}

func (dix *Dix) providerState(p *providerFn) string {
	if p == nil {
		return ""
	}
	stat, ok := dix.providerStats[p.fn]
	if !ok {
		return ""
	}
	if stat.LastError != "" {
		return "error"
	}
	if dix.option.SlowProviderThreshold > 0 && stat.LastDuration > dix.option.SlowProviderThreshold {
		return "slow"
	}
	return ""
}

// ModuleInfo 是模块(pkg)级聚合视图的一行。
type ModuleInfo struct {
	Name          string   `json:"name"`
	TypeCount     int      `json:"type_count"`
	ProviderCount int      `json:"provider_count"`
	ObjectCount   int      `json:"object_count"`
	DependsOn     []string `json:"depends_on,omitempty"`
}

// ModuleGraph 按模块聚合节点,并从声明边提取跨模块依赖(去重、排序)。
func (dix *Dix) ModuleGraph() []ModuleInfo {
	g := dix.graph
	g.mu.RLock()
	defer g.mu.RUnlock()

	pkgOf := func(n Node) string {
		if n.Pkg == "" {
			return "(anonymous)"
		}
		return n.Pkg
	}

	byModule := make(map[string]*ModuleInfo)
	order := make([]string, 0, 8)
	get := func(name string) *ModuleInfo {
		mi, ok := byModule[name]
		if !ok {
			mi = &ModuleInfo{Name: name}
			byModule[name] = mi
			order = append(order, name)
		}
		return mi
	}
	for _, n := range g.nodes {
		mi := get(pkgOf(n))
		switch n.Kind {
		case NodeType:
			mi.TypeCount++
		case NodeProvider:
			mi.ProviderCount++
		case NodeObject:
			mi.ObjectCount++
		}
	}

	containsStr := func(list []string, v string) bool {
		for _, s := range list {
			if s == v {
				return true
			}
		}
		return false
	}
	for _, e := range g.eIndex {
		if e.Kind != EdgeDeclared {
			continue
		}
		fromPkg := pkgOf(g.nodes[e.From])
		toPkg := pkgOf(g.nodes[e.To])
		if fromPkg == toPkg {
			continue
		}
		mi := get(fromPkg)
		if !containsStr(mi.DependsOn, toPkg) {
			mi.DependsOn = append(mi.DependsOn, toPkg)
		}
	}

	out := make([]ModuleInfo, 0, len(order))
	for _, name := range order {
		mi := byModule[name]
		sort.Strings(mi.DependsOn)
		out = append(out, *mi)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// GraphEdge 是邻域子图里的一条声明依赖边(类型 label 表示)。
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// GraphView 是邻域子图:节点摘要 + 声明边。
type GraphView struct {
	Nodes []SearchHit `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// EgoGraph 以 center 类型为中心,沿声明边做 depth 跳 BFS:
// direction=deps 只看依赖方向,dependents 只看被依赖方向,both 双向。
func (dix *Dix) EgoGraph(center string, depth int, direction string) GraphView {
	if depth <= 0 {
		depth = 2
	}
	if depth > 10 {
		depth = 10
	}
	switch direction {
	case "deps", "dependents", "both":
	default:
		direction = "both"
	}

	g := dix.graph
	g.mu.RLock()

	type dEdge struct {
		from, to reflect.Type
	}
	var edges []dEdge
	deps := make(map[reflect.Type][]reflect.Type)
	dependents := make(map[reflect.Type][]reflect.Type)
	var centerType reflect.Type
	for _, n := range g.nodes {
		if n.Kind == NodeType && n.Label == center {
			centerType = n.Type
			break
		}
	}
	if centerType == nil {
		g.mu.RUnlock()
		return GraphView{Nodes: []SearchHit{}, Edges: []GraphEdge{}}
	}
	for _, e := range g.eIndex {
		if e.Kind != EdgeDeclared {
			continue
		}
		from, to := g.nodes[e.From].Type, g.nodes[e.To].Type
		edges = append(edges, dEdge{from: from, to: to})
		deps[from] = append(deps[from], to)
		dependents[to] = append(dependents[to], from)
	}
	g.mu.RUnlock()

	seen := map[reflect.Type]bool{centerType: true}
	frontier := []reflect.Type{centerType}
	for d := 0; d < depth && len(frontier) > 0; d++ {
		var next []reflect.Type
		for _, t := range frontier {
			if direction == "deps" || direction == "both" {
				next = append(next, deps[t]...)
			}
			if direction == "dependents" || direction == "both" {
				next = append(next, dependents[t]...)
			}
		}
		frontier2 := frontier[:0]
		for _, t := range next {
			if !seen[t] {
				seen[t] = true
				frontier2 = append(frontier2, t)
			}
		}
		frontier = frontier2
	}

	label := func(t reflect.Type) string { return t.String() }
	view := GraphView{Nodes: []SearchHit{}, Edges: []GraphEdge{}}
	for _, e := range edges {
		if !seen[e.from] || !seen[e.to] {
			continue
		}
		view.Edges = append(view.Edges, GraphEdge{From: label(e.from), To: label(e.to)})
	}
	for t := range seen {
		view.Nodes = append(view.Nodes, SearchHit{Kind: "type", Label: label(t), Pkg: resolveTypePkgPath(t), State: "instantiated"})
	}
	sort.Slice(view.Nodes, func(i, j int) bool { return view.Nodes[i].Label < view.Nodes[j].Label })
	sort.Slice(view.Edges, func(i, j int) bool {
		if view.Edges[i].From != view.Edges[j].From {
			return view.Edges[i].From < view.Edges[j].From
		}
		return view.Edges[i].To < view.Edges[j].To
	})
	return view
}

// ResolvedCount 是类型维度的解析热度。
type ResolvedCount struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

// ResolvedTopN 返回解析次数最多的前 n 个类型(降序)。
func (dix *Dix) ResolvedTopN(n int) []ResolvedCount {
	if n <= 0 {
		n = 10
	}
	g := dix.graph
	g.mu.RLock()
	defer g.mu.RUnlock()
	counts := make([]ResolvedCount, 0, 8)
	for _, e := range g.eIndex {
		if e.Kind == EdgeResolved && e.Count > 0 {
			counts = append(counts, ResolvedCount{Type: g.nodes[e.To].Type.String(), Count: e.Count})
		}
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].Count != counts[j].Count {
			return counts[i].Count > counts[j].Count
		}
		return counts[i].Type < counts[j].Type
	})
	if len(counts) > n {
		counts = counts[:n]
	}
	return counts
}

// ProblemProviders 返回慢 provider 与错误 provider 的函数名(去重、排序)。
func (dix *Dix) ProblemProviders() (slow, errored []string) {
	slowSet := make(map[string]bool)
	errSet := make(map[string]bool)
	for _, stat := range dix.providerStats {
		if stat.LastError != "" {
			errSet[stat.FunctionName] = true
			continue
		}
		if dix.option.SlowProviderThreshold > 0 && stat.LastDuration > dix.option.SlowProviderThreshold {
			slowSet[stat.FunctionName] = true
		}
	}
	for name := range slowSet {
		slow = append(slow, name)
	}
	for name := range errSet {
		errored = append(errored, name)
	}
	sort.Strings(slow)
	sort.Strings(errored)
	return slow, errored
}
