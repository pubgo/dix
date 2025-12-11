package dixrender

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// DotRenderer implements DOT format graph rendering
type DotRenderer struct {
	Buf    *bytes.Buffer // Exported for testing
	indent string
	cache  map[string]string
}

func NewDotRenderer() *DotRenderer {
	return &DotRenderer{
		Buf:    &bytes.Buffer{},
		indent: "",
		cache:  make(map[string]string),
	}
}

// Writef writes a formatted string to the renderer buffer
func (d *DotRenderer) Writef(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(d.Buf, d.indent+format+"\n", args...)
}

func (d *DotRenderer) writef(format string, args ...interface{}) {
	d.Writef(format, args...)
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
	return d.Buf.String()
}

// FormatAttrs formats attributes map into DOT format string
func (d *DotRenderer) FormatAttrs(attrs map[string]string) string {
	return d.formatAttrs(attrs)
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

// Graph represents dependency graphs in DOT format
type Graph struct {
	Objects       string `json:"objects"`
	Providers     string `json:"providers"`
	ProviderTypes string `json:"provider_types"`
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

// ShouldIncludeType checks if a type should be included in the graph based on filters
func (opts *GraphOptions) ShouldIncludeType(typ string) bool {
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
