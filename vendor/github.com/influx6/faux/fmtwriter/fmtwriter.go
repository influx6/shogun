package fmtwriter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	gexec "os/exec"

	"github.com/influx6/faux/exec"
	"github.com/influx6/faux/metrics"
)

// WriterTo defines a takes the contents of a provided io.WriterTo
// against go fmt and returns the result.
type WriterTo struct {
	io.WriterTo
	goimport        bool
	attemptFallback bool
	Metrics         metrics.Metrics
}

// New returns a new instance of FmtWriterTo.
func New(wt io.WriterTo, useGoImports bool, attemptFallbackToFmtInError bool) *WriterTo {
	return &WriterTo{WriterTo: wt, goimport: useGoImports, attemptFallback: attemptFallbackToFmtInError, Metrics: metrics.New()}
}

// NewWith returns a new instance of FmtWriterTo.
func NewWith(m metrics.Metrics, wt io.WriterTo, useGoImports bool, attemptFallbackToFmtInError bool) *WriterTo {
	return &WriterTo{WriterTo: wt, goimport: useGoImports, attemptFallback: attemptFallbackToFmtInError, Metrics: m}
}

// WriteTo writes the content of the source after running against gofmt to the
// provider writer.
func (fm WriterTo) WriteTo(w io.Writer) (int64, error) {
	var backinput, input, inout, inerr bytes.Buffer

	if fm.Metrics == nil {
		fm.Metrics = metrics.New()
	}

	if n, err := fm.WriterTo.WriteTo(io.MultiWriter(&input, &backinput)); err != nil && err != io.EOF {
		return n, err
	}

	cmdName := "gofmt"
	if fm.goimport {
		if _, err := gexec.LookPath("goimports"); err == nil {
			cmdName = "goimports"
		} else {
			cmdName = "gofmt"
		}
	}

	cmd := exec.New(
		exec.Command(cmdName),
		exec.Input(&input),
		exec.Output(&inout),
		exec.Err(&inerr),
	)

	if err := cmd.Exec(context.Background(), fm.Metrics); err != nil {

		// If we must attempt to fallback to gofmt, due to goimport error, attempt to
		if fm.goimport && fm.attemptFallback {
			fmt.Printf("------------------------- ATTEMPTING GOFMT FALLBACK (GOIMPORTS FAILED) --------------------------------------------\n")
			fmt.Printf("Error:\n%s\n", err.Error())
			fmt.Printf("---------------------------------------------------------------------\n")
			fmt.Printf("StdError:\n%s\n", inerr.String())
			fmt.Printf("---------------------------------------------------------------------\n")
			return (WriterTo{WriterTo: &backinput}).WriteTo(w)
		}

		errcount, _ := inerr.WriteTo(w)
		linecount, _ := fmt.Fprintf(w, "\n-----------------------\n")
		outcount, _ := backinput.WriteTo(w)

		return (errcount + int64(linecount) + outcount), fmt.Errorf("GoFmt Error: %+q (See generated file for fmt Error)", err)
	}

	return inout.WriteTo(w)
}
