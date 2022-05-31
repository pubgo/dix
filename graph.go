package dix

import (
	"bytes"
	"fmt"
	"io"
)

func fPrintln(writer io.Writer, msg string) {
	_, _ = fmt.Fprintln(writer, msg)
}

func (x *dix) graph() string {
	b := &bytes.Buffer{}
	fPrintln(b, "digraph G {")
	fPrintln(b, "\tsubgraph providers {")
	fPrintln(b, "\t\tlabel=providers")
	for k, vs := range x.providers {
		for _, n := range vs {
			fn := callerWithFunc(n.fn)
			fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> "%s"`, fn, k))
			for _, in := range n.input {
				fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> "%s"`, in.typ, fn))
			}
		}
	}
	fPrintln(b, "\t}")

	fPrintln(b, "\tsubgraph invokes {")
	fPrintln(b, "\t\tlabel=invokes")
	for _, n := range x.invokes {
		fn := callerWithFunc(n.fn)
		for _, in := range n.input {
			fPrintln(b, fmt.Sprintf("\t\t"+`"%s" -> "%s"`, in.typ, fn))
		}
	}
	fPrintln(b, "\t}")
	fPrintln(b, "}")

	return b.String()
}
