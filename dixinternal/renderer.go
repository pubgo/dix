package dixinternal

import (
	"bytes"
	"fmt"

	"github.com/pubgo/funk/stack"
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

	var result bytes.Buffer
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

func (x *Dix) providerGraph() string {
	d := NewDotRenderer()
	d.writef("digraph G {")
	d.BeginSubgraph("cluster_providers", "providers")

	for providerOutputType, nodes := range x.providers {
		for _, n := range nodes {
			fn := stack.CallerWithFunc(n.fn).String()
			d.RenderEdge(fn, providerOutputType.String(), nil)
			for _, in := range n.inputList {
				d.RenderEdge(in.typ.String(), fn, nil)
			}
		}
	}

	d.EndSubgraph()
	d.writef("}")
	return d.String()
}

func (x *Dix) objectGraph() string {
	d := NewDotRenderer()
	d.writef("digraph G {")
	d.BeginSubgraph("cluster_objects", "objects")

	for k, objects := range x.objects {
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
