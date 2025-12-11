package dixrender

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/samber/lo"
)

// ProviderFnAccessor provides access to provider function information
type ProviderFnAccessor interface {
	GetFn() reflect.Value
	GetInputTypes() []reflect.Type
}

// DixAccessor provides access to Dix container data for rendering
// This interface allows dixrender to work without importing dixinternal
type DixAccessor interface {
	GetProviders() map[reflect.Type][]ProviderFnAccessor
	GetObjects() map[reflect.Type]map[string][]reflect.Value
	GetProviderInputTypes(p ProviderFnAccessor) []reflect.Type
	GetProvideAllInputTypes(typ reflect.Type) []reflect.Type
	GetFnName(fn reflect.Value) string
	GetProviderFn(p ProviderFnAccessor) reflect.Value
}

// GenerateProviderGraphTypes generates a graph showing type dependencies
func GenerateProviderGraphTypes(dix DixAccessor, opts *GraphOptions) string {
	d := NewDotRenderer()
	d.Writef("digraph G {")
	d.Writef(`
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

	providers := dix.GetProviders()

	// Track visited nodes to avoid duplication
	visitedNodes := make(map[string]bool)

	// First pass: collect all nodes and build dependency hierarchy
	typeNodeMap := make(map[string]int) // type -> level
	var allTypes []string

	for providerOutputType, nodes := range providers {
		if providerOutputType.String() == "" {
			continue
		}

		if !shouldIncludeType(opts, providerOutputType.String()) {
			continue
		}

		if !visitedNodes[providerOutputType.String()] {
			allTypes = append(allTypes, providerOutputType.String())
			visitedNodes[providerOutputType.String()] = true
			typeNodeMap[providerOutputType.String()] = 0 // Root level
		}

		for _, n := range nodes {
			inputTypes := dix.GetProviderInputTypes(n)
			for _, inType := range inputTypes {
				var typesToRender []reflect.Type
				if inType.Kind() == reflect.Struct {
					typesToRender = lo.Uniq(dix.GetProvideAllInputTypes(inType))
				} else {
					typesToRender = []reflect.Type{inType}
				}

				for _, t := range typesToRender {
					if t.String() != "" && shouldIncludeType(opts, t.String()) {
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
		for providerOutputType, nodes := range providers {
			if providerOutputType.String() == "" || !shouldIncludeType(opts, providerOutputType.String()) {
				continue
			}

			currentLevel := typeNodeMap[providerOutputType.String()]

			for _, n := range nodes {
				inputTypes := dix.GetProviderInputTypes(n)
				for _, inType := range inputTypes {
					var typesToRender []reflect.Type
					if inType.Kind() == reflect.Struct {
						typesToRender = lo.Uniq(dix.GetProvideAllInputTypes(inType))
					} else {
						typesToRender = []reflect.Type{inType}
					}

					for _, t := range typesToRender {
						if t.String() != "" && shouldIncludeType(opts, t.String()) {
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
	for providerOutputType, nodes := range providers {
		if providerOutputType.String() == "" || !shouldIncludeType(opts, providerOutputType.String()) {
			continue
		}

		// Check if this node exists in our filtered map
		if _, exists := typeNodeMap[providerOutputType.String()]; !exists {
			continue
		}

		for _, n := range nodes {
			inputTypes := dix.GetProviderInputTypes(n)
			for _, inType := range inputTypes {
				var typesToRender []reflect.Type
				if inType.Kind() == reflect.Struct && opts.ShowStructFields {
					typesToRender = lo.Uniq(dix.GetProvideAllInputTypes(inType))
				} else {
					typesToRender = []reflect.Type{inType}
				}

				for _, t := range typesToRender {
					if t.String() != "" && shouldIncludeType(opts, t.String()) {
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
	d.Writef("}")
	return d.String()
}

// GenerateProviderGraph generates a graph showing provider functions
func GenerateProviderGraph(dix DixAccessor, opts *GraphOptions) string {
	d := NewDotRenderer()
	d.Writef("digraph G {")
	d.Writef(`
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

	providers := dix.GetProviders()

	// Track visited nodes
	visitedNodes := make(map[string]bool)
	functionNodes := make(map[string]bool)

	// Collect all function nodes and their dependencies
	for providerOutputType, nodes := range providers {
		for _, n := range nodes {
			fnName := dix.GetFnName(dix.GetProviderFn(n))
			fn := filepath.Base(fnName)

			if !shouldIncludeType(opts, providerOutputType.String()) {
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
			inputTypes := dix.GetProviderInputTypes(n)
			for _, inType := range inputTypes {
				if inType.Kind() == reflect.Struct && opts.ShowStructFields {
					inTypes := lo.Uniq(dix.GetProvideAllInputTypes(inType))
					for _, inType := range inTypes {
						if shouldIncludeType(opts, inType.String()) {
							if !visitedNodes[inType.String()] {
								d.RenderNode(inType.String(), nil)
								visitedNodes[inType.String()] = true
							}
							d.RenderEdge(inType.String(), fn, nil)
						}
					}
				} else {
					if shouldIncludeType(opts, inType.String()) {
						if !visitedNodes[inType.String()] {
							d.RenderNode(inType.String(), nil)
							visitedNodes[inType.String()] = true
						}
						d.RenderEdge(inType.String(), fn, nil)
					}
				}
			}
		}
	}

	d.EndSubgraph()
	d.Writef("}")
	return d.String()
}

// GenerateObjectGraph generates a graph showing object instances
func GenerateObjectGraph(dix DixAccessor) string {
	d := NewDotRenderer()
	d.Writef("digraph G {")
	d.BeginSubgraph("cluster_objects", "objects")

	objects := dix.GetObjects()

	for k, objectsMap := range objects {
		for g, values := range objectsMap {
			for _, v := range values {
				d.RenderEdge(k.String(), fmt.Sprintf("%s -> %s", g, v.Type().String()), nil)
			}
		}
	}

	d.EndSubgraph()
	d.Writef("}")
	return d.String()
}

// shouldIncludeType checks if a type should be included in the graph based on filters
func shouldIncludeType(opts *GraphOptions, typ string) bool {
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

// getPackageName extracts package name from a full type string
func getPackageName(fullTypeName string) string {
	if idx := strings.LastIndex(fullTypeName, "."); idx != -1 {
		return fullTypeName[:idx]
	}
	return ""
}

// GenerateGraph generates a complete dependency graph with default options
func GenerateGraph(dix DixAccessor) *Graph {
	return GenerateGraphWithOptions(dix, NewGraphOptions())
}

// GenerateGraphWithOptions generates a complete dependency graph with custom options
func GenerateGraphWithOptions(dix DixAccessor, opts *GraphOptions) *Graph {
	return &Graph{
		Objects:       GenerateObjectGraph(dix),
		Providers:     GenerateProviderGraph(dix, opts),
		ProviderTypes: GenerateProviderGraphTypes(dix, opts),
	}
}
