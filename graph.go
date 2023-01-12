package dix

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
	for k, vs := range x.providers {
		for _, n := range vs {
			fn := stack.CallerWithFunc(n.fn).String()
			fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> "%s"`, fn, k))
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
		for g, v := range objects {
			fPrintln(b, fmt.Sprintf("\t\t"+`object -> "%s" -> "%s" -> %v`, k, g, v))
		}
	}
	fPrintln(b, "\t}")
	fPrintln(b, "}")
	return b.String()
}
