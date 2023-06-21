package dix_inter

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pubgo/funk/stack"
)

func fPrintln(writer io.Writer, msg string) {
	_, _ = fmt.Fprintln(writer, msg)
}

func (x *Dix) providerGraph() string {
	b := &bytes.Buffer{}
	fPrintln(b, "digraph G {")

	fPrintln(b, "\tsubgraph providers {")
	fPrintln(b, "\t\tlabel=providers")
	for providerOutputType, nodes := range x.providers {
		for _, n := range nodes {
			fn := stack.CallerWithFunc(n.fn).String()
			fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> "%s"`, fn, providerOutputType))
			for _, in := range n.input {
				fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> "%s"`, in.typ, fn))
			}
		}
	}
	fPrintln(b, "\t}")

	fPrintln(b, "}")
	return b.String()
}

func (x *Dix) objectGraph() string {
	b := &bytes.Buffer{}
	fPrintln(b, "digraph G {")
	fPrintln(b, "\tsubgraph objects {")
	fPrintln(b, "\t\tlabel=objects")
	for k, objects := range x.objects {
		for g, values := range objects {
			var data []string
			for i := range values {
				data = append(data, fmt.Sprintf("%#v", values[i].Interface()))
			}
			fPrintln(b, fmt.Sprintf("\t\t"+`object -> "%s" -> "%s" -> %v`, k, g, data))
		}
	}
	fPrintln(b, "\t}")
	fPrintln(b, "}")
	return b.String()
}
