package dixinternal

import (
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

// NodeID 是 Graph 内节点自增编号。
type NodeID uint32

// NodeKind 区分依赖类型节点、provider 节点与已创建对象节点。
type NodeKind uint8

const (
	NodeType NodeKind = iota
	NodeProvider
	NodeObject
)

// EdgeKind 区分声明依赖、产物输出与运行时实际解析。
type EdgeKind uint8

const (
	EdgeDeclared EdgeKind = iota
	EdgeProduced
	EdgeResolved
)

// Node 是 Graph 节点。Object 节点的存在即"已实例化"状态。
type Node struct {
	ID       NodeID
	Kind     NodeKind
	Type     reflect.Type
	Group    string
	Provider *providerFn
	Label    string
	Pkg      string
}

// Edge 是 Graph 有向边。声明边:输出类型 -> 输入类型;
// 产物边:provider -> 输出类型;解析边:provider -> 输出类型(执行计数)。
type Edge struct {
	From, To  NodeID
	Kind      EdgeKind
	Field     string
	Aggregate string
	Provider  *providerFn
	Count     int64
}

type nodeKey struct {
	kind  NodeKind
	typ   reflect.Type
	group string
	fn    reflect.Value // provider 节点:函数值(handleProvide 对 struct 输出按字段递归,指针每次新建,函数值才稳定可比较)
}

type edgeKey struct {
	from, to NodeID
	kind     EdgeKind
	field    string
	agg      string
	fn       reflect.Value
}

// Graph 是容器运行时依赖图。Dix 单线程写,但 dixhttp 并发读,
// 因此全部访问走读写锁;version 供快照判脏。
type Graph struct {
	mu      sync.RWMutex
	nodes   []Node
	nIndex  map[nodeKey]NodeID
	eIndex  map[edgeKey]*Edge
	version atomic.Uint64
}

func newGraph() *Graph {
	return &Graph{
		nIndex: make(map[nodeKey]NodeID),
		eIndex: make(map[edgeKey]*Edge),
	}
}

// Version 返回图版本号;Provide 或新对象入图时递增。
func (g *Graph) Version() uint64 { return g.version.Load() }

func (g *Graph) bumpVersion() { g.version.Add(1) }

// searchNodes 按关键字(小写包含)检索节点 Label 与 provider 函数名。
// P4 定义查询形状后可升级为倒排索引;当前线性扫描,规模内足够。
func (g *Graph) searchNodes(q string, limit int) []Node {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return nil
	}
	if limit <= 0 {
		limit = 50
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	var out []Node
	for _, n := range g.nodes {
		hay := strings.ToLower(n.Label)
		if n.Provider != nil {
			hay += " " + strings.ToLower(GetFnName(n.Provider.fn))
		}
		if strings.Contains(hay, q) {
			out = append(out, n)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

func (g *Graph) providerNode(p *providerFn, outTyp reflect.Type) NodeID {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.nodeLocked(NodeProvider, outTyp, "", p.fn)
}

// node 取得或创建类型节点。
func (g *Graph) node(kind NodeKind, typ reflect.Type, group string, provider *providerFn) NodeID {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.nodeLocked(kind, typ, group, fnKeyOf(provider))
}

// addProduced 记录 provider 产物边。
func (g *Graph) addProduced(pNode NodeID, outTyp reflect.Type) {
	g.mu.Lock()
	defer g.mu.Unlock()
	to := g.nodeLocked(NodeType, outTyp, "", reflect.Value{})
	g.edgeLocked(pNode, to, EdgeProduced, "", "", nil)
}

// addDeclared 记录声明依赖边:输出类型 -> 输入类型。
func (g *Graph) addDeclared(outTyp, inTyp reflect.Type, agg string, p *providerFn) {
	g.mu.Lock()
	defer g.mu.Unlock()
	from := g.nodeLocked(NodeType, outTyp, "", reflect.Value{})
	to := g.nodeLocked(NodeType, inTyp, "", reflect.Value{})
	g.edgeLocked(from, to, EdgeDeclared, "", agg, p)
}

// markResolved 累加 provider 实际执行计数。
func (g *Graph) markResolved(pNode NodeID, outTyp reflect.Type) {
	g.mu.Lock()
	defer g.mu.Unlock()
	to := g.nodeLocked(NodeType, outTyp, "", reflect.Value{})
	g.edgeLocked(pNode, to, EdgeResolved, "", "", nil).Count++
}

// addObject 记录 (type, group) 对象节点;返回是否新建。
func (g *Graph) addObject(typ reflect.Type, group string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	key := nodeKey{kind: NodeObject, typ: typ, group: group}
	if _, ok := g.nIndex[key]; ok {
		return false
	}
	g.nodeLocked(NodeObject, typ, group, reflect.Value{})
	return true
}

// declaredAdjacency 投影声明边为 outputType -> inputTypes 邻接表,
// 语义与原 buildDependencyGraph 完全一致,供环检测使用。
func (g *Graph) declaredAdjacency() map[reflect.Type]map[reflect.Type]bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	adj := make(map[reflect.Type]map[reflect.Type]bool)
	for _, e := range g.eIndex {
		if e.Kind != EdgeDeclared {
			continue
		}
		out := g.nodes[e.From].Type
		in := g.nodes[e.To].Type
		if adj[out] == nil {
			adj[out] = make(map[reflect.Type]bool)
		}
		adj[out][in] = true
	}
	return adj
}

func (g *Graph) nodeLocked(kind NodeKind, typ reflect.Type, group string, fn reflect.Value) NodeID {
	key := nodeKey{kind: kind, typ: typ, group: group, fn: fn}
	if id, ok := g.nIndex[key]; ok {
		return id
	}
	provider := (*providerFn)(nil)
	if fn.IsValid() {
		provider = &providerFn{fn: fn}
	}
	id := NodeID(len(g.nodes))
	g.nodes = append(g.nodes, Node{
		ID: id, Kind: kind, Type: typ, Group: group, Provider: provider,
		Label: typ.String(), Pkg: resolveTypePkgPath(typ),
	})
	g.nIndex[key] = id
	return id
}

// fnKeyOf 取 providerFn 的函数值作为稳定键;nil/零值安全。
func fnKeyOf(p *providerFn) reflect.Value {
	if p == nil {
		return reflect.Value{}
	}
	return p.fn
}

func (g *Graph) edgeLocked(from, to NodeID, kind EdgeKind, field, agg string, p *providerFn) *Edge {
	key := edgeKey{from: from, to: to, kind: kind, field: field, agg: agg, fn: fnKeyOf(p)}
	if e, ok := g.eIndex[key]; ok {
		return e
	}
	e := &Edge{From: from, To: to, Kind: kind, Field: field, Aggregate: agg, Provider: p}
	g.eIndex[key] = e
	return e
}
