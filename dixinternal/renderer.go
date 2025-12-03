package dixinternal

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/samber/lo"
)

// DotRenderer implements DOT format graph rendering
type DotRenderer struct {
	buf    *bytes.Buffer
	indent string
	cache  map[string]string
}

func NewDotRenderer() *DotRenderer {
	return &DotRenderer{
		buf:    &bytes.Buffer{},
		indent: "",
		cache:  make(map[string]string),
	}
}

func (d *DotRenderer) writef(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(d.buf, d.indent+format+"\n", args...)
}

func (d *DotRenderer) RenderNode(name string, attrs map[string]string) {
	d.writef("%s [label=\"%s\"%s]", name, name, d.formatAttrs(attrs))
}

func (d *DotRenderer) RenderEdge(from, to string, attrs map[string]string) {
	d.writef(`"%s" -> "%s" %s`, from, to, d.formatAttrs(attrs))
}

func (d *DotRenderer) BeginSubgraph(name, label string) {
	d.writef("subgraph %s {", name)
	d.indent += "\t"
	d.writef("label=\"%s\"", label)
}

func (d *DotRenderer) EndSubgraph() {
	d.indent = d.indent[:len(d.indent)-1]
	d.writef("}")
}

func (d *DotRenderer) String() string {
	return d.buf.String()
}

func (d *DotRenderer) formatAttrs(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}

	// Sort keys to ensure consistent ordering
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var result bytes.Buffer
	result.WriteString(" [")
	first := true
	for _, k := range keys {
		if !first {
			result.WriteString(",")
		}
		first = false
		v := attrs[k]
		fmt.Fprintf(&result, "%s=\"%s\"", k, v)
	}
	result.WriteString("]")
	return result.String()
}

// GraphOptions holds configuration options for graph rendering
type GraphOptions struct {
	// MaxDepth limits the depth of dependencies to show (0 = unlimited)
	MaxDepth int

	// GroupByPackage enables grouping nodes by package
	GroupByPackage bool

	// ShowStructFields controls whether to show struct field dependencies
	ShowStructFields bool

	// FilterPackages allows filtering by specific packages
	FilterPackages []string
}

// NewGraphOptions creates GraphOptions with sensible defaults
func NewGraphOptions() *GraphOptions {
	return &GraphOptions{
		MaxDepth:         0, // Unlimited by default
		GroupByPackage:   true,
		ShowStructFields: false,
		FilterPackages:   []string{},
	}
}

// shouldIncludeType checks if a type should be included in the graph based on filters
func (opts *GraphOptions) shouldIncludeType(typ string) bool {
	if len(opts.FilterPackages) == 0 {
		return true
	}

	for _, pkg := range opts.FilterPackages {
		if strings.Contains(typ, pkg) {
			return true
		}
	}
	return false
}

// getNodeName extracts a clean node name, optionally grouped by package
func (opts *GraphOptions) getNodeName(typ string, groupByPackage bool) string {
	if !groupByPackage {
		return typ
	}

	// Extract package name for grouping
	parts := strings.Split(typ, ".")
	if len(parts) > 1 {
		// Return package as cluster name and type as node name
		return strings.Join(parts[:len(parts)-1], ".") + "." + parts[len(parts)-1]
	}
	return typ
}

func (dix *Dix) providerGraphTypes() string {
	return dix.providerGraphTypesWithOptions(NewGraphOptions())
}

func (dix *Dix) providerGraphTypesWithOptions(opts *GraphOptions) string {
	d := NewDotRenderer()
	d.writef("digraph G {")
	d.writef(`
	// 设置布局引擎
    layout=dot;

    // 图形整体设置
    rankdir=TB;          // 顶部到底部布局，更适合层次结构
    overlap=false;       // 避免节点重叠
    splines=true;        // 使用曲线边
    nodesep=0.8;         // 增加节点间距
    ranksep=1.2;         // 增加层级间距
    concentrate=true;    // 合并重复边

    // 节点样式
    node [
        shape=box,
        style=filled,
        fillcolor="#F9F9F9",
        color="#666666",
        fontsize=10,
        fontname="Arial",
        width=0.2,
        height=0.2,
        fixedsize=false
    ];

    // 边样式
    edge [
        arrowhead=vee,
        arrowsize=0.6,
        color="#888888",
        penwidth=0.8
    ];
`)
	d.BeginSubgraph("cluster_providers", "providers")

	// Track visited nodes to avoid duplication
	visitedNodes := make(map[string]bool)

	// First pass: collect all nodes and build dependency hierarchy
	typeNodeMap := make(map[string]int) // type -> level
	var allTypes []string

	for providerOutputType, nodes := range dix.providers {
		if providerOutputType.String() == "" {
			continue
		}

		if !opts.shouldIncludeType(providerOutputType.String()) {
			continue
		}

		if !visitedNodes[providerOutputType.String()] {
			allTypes = append(allTypes, providerOutputType.String())
			visitedNodes[providerOutputType.String()] = true
			typeNodeMap[providerOutputType.String()] = 0 // Root level
		}

		for _, n := range nodes {
			for _, in := range n.inputList {
				var typesToRender []reflect.Type
				if in.typ.Kind() == reflect.Struct {
					typesToRender = lo.Uniq(lo.Map(getProvideAllInputs(in.typ), func(item *providerInputType, index int) reflect.Type { return item.typ }))
				} else {
					typesToRender = []reflect.Type{in.typ}
				}

				for _, t := range typesToRender {
					if t.String() != "" && opts.shouldIncludeType(t.String()) {
						if !visitedNodes[t.String()] {
							allTypes = append(allTypes, t.String())
							visitedNodes[t.String()] = true
						}
					}
				}
			}
		}
	}

	// Calculate dependency levels
	for i := 0; i < 10; i++ { // Limit iterations to prevent infinite loops
		changed := false
		for providerOutputType, nodes := range dix.providers {
			if providerOutputType.String() == "" || !opts.shouldIncludeType(providerOutputType.String()) {
				continue
			}

			currentLevel := typeNodeMap[providerOutputType.String()]

			for _, n := range nodes {
				for _, in := range n.inputList {
					var typesToRender []reflect.Type
					if in.typ.Kind() == reflect.Struct {
						typesToRender = lo.Uniq(lo.Map(getProvideAllInputs(in.typ), func(item *providerInputType, index int) reflect.Type { return item.typ }))
					} else {
						typesToRender = []reflect.Type{in.typ}
					}

					for _, t := range typesToRender {
						if t.String() != "" && opts.shouldIncludeType(t.String()) {
							// Dependencies should be at a lower level (higher number)
							if typeNodeMap[t.String()] <= currentLevel {
								typeNodeMap[t.String()] = currentLevel + 1
								changed = true
							}
						}
					}
				}
			}
		}
		if !changed {
			break
		}
	}

	// Apply max depth filter
	if opts.MaxDepth > 0 {
		for typ, level := range typeNodeMap {
			if level >= opts.MaxDepth {
				delete(typeNodeMap, typ)
			}
		}
	}

	// Group nodes by package if enabled
	if opts.GroupByPackage {
		packageClusters := make(map[string][]string)
		for typ := range typeNodeMap {
			packageName := getPackageName(typ)
			if packageName != "" {
				packageClusters[packageName] = append(packageClusters[packageName], typ)
			}
		}

		// Create clusters for each package
		for pkgName, types := range packageClusters {
			if len(types) > 1 { // Only create cluster if more than one type
				clusterName := fmt.Sprintf("cluster_%s", strings.ReplaceAll(pkgName, ".", "_"))
				d.BeginSubgraph(clusterName, pkgName)
				for _, typ := range types {
					shortName := typ[strings.LastIndex(typ, ".")+1:]
					d.RenderNode(shortName, map[string]string{"tooltip": typ})
				}
				d.EndSubgraph()
			} else {
				// Single type, render normally
				for _, typ := range types {
					d.RenderNode(typ, nil)
				}
			}
		}
	} else {
		// Render all nodes without clustering
		for typ := range typeNodeMap {
			d.RenderNode(typ, nil)
		}
	}

	// Render edges
	for providerOutputType, nodes := range dix.providers {
		if providerOutputType.String() == "" || !opts.shouldIncludeType(providerOutputType.String()) {
			continue
		}

		// Check if this node exists in our filtered map
		if _, exists := typeNodeMap[providerOutputType.String()]; !exists {
			continue
		}

		for _, n := range nodes {
			for _, in := range n.inputList {
				var typesToRender []reflect.Type
				if in.typ.Kind() == reflect.Struct && opts.ShowStructFields {
					typesToRender = lo.Uniq(lo.Map(getProvideAllInputs(in.typ), func(item *providerInputType, index int) reflect.Type { return item.typ }))
				} else {
					typesToRender = []reflect.Type{in.typ}
				}

				for _, t := range typesToRender {
					if t.String() != "" && opts.shouldIncludeType(t.String()) {
						// Check if this dependency node exists in our filtered map
						if _, depExists := typeNodeMap[t.String()]; depExists {
							if opts.GroupByPackage {
								fromName := t.String()[strings.LastIndex(t.String(), ".")+1:]
								toName := providerOutputType.String()[strings.LastIndex(providerOutputType.String(), ".")+1:]
								d.RenderEdge(fromName, toName, nil)
							} else {
								d.RenderEdge(t.String(), providerOutputType.String(), nil)
							}
						}
					}
				}
			}
		}
	}

	d.EndSubgraph()
	d.writef("}")
	return d.String()
}

func (dix *Dix) providerGraph() string {
	return dix.providerGraphWithOptions(NewGraphOptions())
}

func (dix *Dix) providerGraphWithOptions(opts *GraphOptions) string {
	d := NewDotRenderer()
	d.writef("digraph G {")
	d.writef(`
	// 设置布局引擎
    layout=dot;

    // 图形整体设置
    rankdir=TB;          // 顶部到底部布局
    overlap=false;       // 避免节点重叠
    splines=true;        // 使用曲线边
    nodesep=0.8;         // 节点间距
    ranksep=1.2;         // 层级间距
    concentrate=true;    // 合并重复边

    // 节点样式
    node [
        shape=box,
        style=filled,
        fillcolor="#F9F9F9",
        color="#666666",
        fontsize=10,
        fontname="Arial",
        width=0.2,
        height=0.2,
        fixedsize=false
    ];

    // 边样式
    edge [
        arrowhead=vee,
        arrowsize=0.6,
        color="#888888",
        penwidth=0.8
    ];
`)
	d.BeginSubgraph("cluster_providers", "providers")

	// Track visited nodes
	visitedNodes := make(map[string]bool)
	functionNodes := make(map[string]bool)

	// Collect all function nodes and their dependencies
	for providerOutputType, nodes := range dix.providers {
		for _, n := range nodes {
			fnName := GetFnName(n.fn)
			fn := filepath.Base(fnName)

			if !opts.shouldIncludeType(providerOutputType.String()) {
				continue
			}

			functionNodes[fn] = true

			if !visitedNodes[providerOutputType.String()] {
				d.RenderNode(providerOutputType.String(), map[string]string{"shape": "ellipse", "fillcolor": "#E8F4FD"})
				visitedNodes[providerOutputType.String()] = true
			}

			// Add edge from function to output type
			d.RenderEdge(fn, providerOutputType.String(), nil)

			// Add edges for input dependencies
			for _, in := range n.inputList {
				if in.typ.Kind() == reflect.Struct && opts.ShowStructFields {
					inTypes := lo.Uniq(lo.Map(getProvideAllInputs(in.typ), func(item *providerInputType, index int) reflect.Type { return item.typ }))
					for _, inType := range inTypes {
						if opts.shouldIncludeType(inType.String()) {
							if !visitedNodes[inType.String()] {
								d.RenderNode(inType.String(), nil)
								visitedNodes[inType.String()] = true
							}
							d.RenderEdge(inType.String(), fn, nil)
						}
					}
				} else {
					if opts.shouldIncludeType(in.typ.String()) {
						if !visitedNodes[in.typ.String()] {
							d.RenderNode(in.typ.String(), nil)
							visitedNodes[in.typ.String()] = true
						}
						d.RenderEdge(in.typ.String(), fn, nil)
					}
				}
			}
		}
	}

	d.EndSubgraph()
	d.writef("}")
	return d.String()
}

func (dix *Dix) objectGraph() string {
	d := NewDotRenderer()
	d.writef("digraph G {")
	d.BeginSubgraph("cluster_objects", "objects")

	for k, objects := range dix.objects {
		for g, values := range objects {
			for _, v := range values {
				d.RenderEdge(k.String(), fmt.Sprintf("%s -> %s", g, v.Type().String()), nil)
			}
		}
	}

	d.EndSubgraph()
	d.writef("}")
	return d.String()
}

// getPackageName extracts package name from a full type string
func getPackageName(fullTypeName string) string {
	if idx := strings.LastIndex(fullTypeName, "."); idx != -1 {
		return fullTypeName[:idx]
	}
	return ""
}
