package dix

import (
	"bytes"
	"fmt"
)

func (x *dix) graph() string {
	b := &bytes.Buffer{}
	fPrintln(b, "digraph G {")
	fPrintln(b, "\tsubgraph cluster_0 {")
	fPrintln(b, "\t\tlabel=providers1")
	for k, vs := range x.providers1 {
		for k1, v1 := range vs {
			for i := range v1 {
				fn := callerWithFunc(v1[i].fn)
				fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> %s -> "%s"`, k, k1, fn))
				for _, v2 := range v1[i].output {
					fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> %s -> "%s" -> "%s"`, k, k1, fn, v2))
				}
			}
		}
	}
	fPrintln(b, "\t}")

	fPrintln(b, "\tsubgraph cluster_2 {")
	fPrintln(b, "\t\tlabel=abc_providers")
	for k, vs := range x.abcProviders {
		for k1, v1 := range vs {
			for i := range v1 {
				fn := callerWithFunc(v1[i].fn)
				fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> %s -> "%s"`, k, k1, fn))
				for _, v2 := range v1[i].output {
					fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> %s -> "%s" -> "%s"`, k, k1, fn, v2))
				}
			}
		}
	}
	fPrintln(b, "\t}")

	fPrintln(b, "\tsubgraph cluster_1 {")
	fPrintln(b, "\t\tlabel=values")
	for k, v := range x.values {
		for k1, v1 := range v {
			fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> %s -> "%s"`, k, k1, v1.String()))
		}
	}
	fPrintln(b, "\t}")

	fPrintln(b, "\tsubgraph cluster_3 {")
	fPrintln(b, "\t\tlabel=abc_values")
	for k, v := range x.abcValues {
		for k1, v1 := range v {
			fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> %s -> "%s"`, k, k1, v1.String()))
		}
	}
	fPrintln(b, "\t}")
	fPrintln(b, "}")

	return b.String()
}

func (x *dix) json() map[string]interface{} {
	var nodes []string
	var values []string
	var abcNodes []string
	var abcValues []string
	for k, vs := range x.providers1 {
		for k1, v1 := range vs {
			for i := range v1 {
				fn := callerWithFunc(v1[i].fn)
				nodes = append(nodes, fmt.Sprintf(`%s -- %s -- %s`, k, k1, fn))
				for _, v2 := range v1[i].output {
					nodes = append(nodes, fmt.Sprintf(`%s -- %s -- %s -- %s`, k, k1, fn, v2))
				}
			}
		}
	}

	for k, v := range x.values {
		for k1, v1 := range v {
			values = append(values, fmt.Sprintf(`%s -- %s -- %s`, k, k1, v1.String()))
		}
	}

	for k, vs := range x.abcProviders {
		for k1, v1 := range vs {
			for i := range v1 {
				fn := callerWithFunc(v1[i].fn)
				abcNodes = append(abcNodes, fmt.Sprintf(`%s -- %s -- %s`, k, k1, fn))
				for _, v2 := range v1[i].output {
					abcNodes = append(abcNodes, fmt.Sprintf(`%s -- %s -- %s -- %s`, k, k1, fn, v2))
				}
			}
		}
	}

	for k, v := range x.abcValues {
		for k1, v1 := range v {
			abcValues = append(abcValues, fmt.Sprintf(`%s -- %s -- %s`, k, k1, v1.String()))
		}
	}

	return map[string]interface{}{
		"nodes":      nodes,
		"values":     values,
		"abc_Nodes":  abcNodes,
		"abc_Values": abcValues,
	}
}
