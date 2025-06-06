package dixinternal

import (
	"fmt"
	"reflect"
	"strings"
)

// DotRenderer DOT格式图形渲染器
type DotRenderer struct {
	buf    strings.Builder
	indent string
}

// NewDotRenderer 创建新的DOT渲染器
func NewDotRenderer() *DotRenderer {
	return &DotRenderer{
		indent: "",
	}
}

func (d *DotRenderer) writef(format string, args ...interface{}) {
	d.buf.WriteString(d.indent)
	d.buf.WriteString(fmt.Sprintf(format, args...))
	d.buf.WriteString("\n")
}

func (d *DotRenderer) RenderNode(name string, attrs map[string]string) {
	d.writef("%s [label=\"%s\"%s]", name, name, d.formatAttrs(attrs))
}

func (d *DotRenderer) RenderEdge(from, to string, attrs map[string]string) {
	d.writef("%s -> %s%s", from, to, d.formatAttrs(attrs))
}

func (d *DotRenderer) BeginSubgraph(name, label string) {
	d.writef("subgraph %s {", name)
	d.indent += "\t"
	d.writef("label=\"%s\"", label)
}

func (d *DotRenderer) EndSubgraph() {
	if len(d.indent) > 0 {
		d.indent = d.indent[:len(d.indent)-1]
	}
	d.writef("}")
}

func (d *DotRenderer) String() string {
	return d.buf.String()
}

func (d *DotRenderer) formatAttrs(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}

	var result strings.Builder
	result.WriteString(" [")
	first := true
	for k, v := range attrs {
		if !first {
			result.WriteString(",")
		}
		first = false
		fmt.Fprintf(&result, "%s=\"%s\"", k, v)
	}
	result.WriteString("]")
	return result.String()
}

// RenderProviders 渲染提供者图
func (d *DotRenderer) RenderProviders(providers map[reflect.Type][]Provider) string {
	d.buf.Reset()
	d.writef("digraph G {")
	d.BeginSubgraph("cluster_providers", "providers")

	for providerType, providerList := range providers {
		for _, provider := range providerList {
			providerName := fmt.Sprintf("provider_%p", provider)
			d.RenderEdge(providerName, providerType.String(), nil)

			for _, dep := range provider.Dependencies() {
				d.RenderEdge(dep.Type().String(), providerName, nil)
			}
		}
	}

	d.EndSubgraph()
	d.writef("}")
	return d.String()
}

// RenderObjects 渲染对象图
func (d *DotRenderer) RenderObjects(objects map[reflect.Type]map[string][]reflect.Value) string {
	d.buf.Reset()
	d.writef("digraph G {")
	d.BeginSubgraph("cluster_objects", "objects")

	for typ, groups := range objects {
		for group, values := range groups {
			for i, val := range values {
				objName := fmt.Sprintf("%s_%s_%d", typ.String(), group, i)
				d.RenderEdge(typ.String(), objName, map[string]string{
					"label": fmt.Sprintf("%s -> %s", group, val.Type().String()),
				})
			}
		}
	}

	d.EndSubgraph()
	d.writef("}")
	return d.String()
}
