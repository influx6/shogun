// Package kensho provides a series of tests case which can be used to validate that a giving
// generated shogun package meet it's design and expected operation.
package kensho

import (
	"bytes"
	gctx "context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/influx6/faux/context"
	"github.com/influx6/faux/tests"
	"github.com/influx6/shogun/internal"
)

// TestWriterFunction validates the behaviour of a function that expects a writer argument.
func TestWriterFunction(fun internal.ShogunFunc) {
	var err error

	defer func() {
		if rec := recover(); rec != nil {
			switch drec := rec.(type) {
			case error:
				err = drec
			default:
				err = fmt.Errorf("Recover Error: %+q", rec)
			}
		}
	}()

	var outgoing bytes.Buffer

	realFunc := fun.Function.(func(io.WriteCloser))
	realGCtxFunc := fun.Function.(func(gctx.Context, io.WriteCloser))
	realFCtxFunc := fun.Function.(func(context.CancelContext, io.WriteCloser))

	realFuncWithReturn := fun.Function.(func(io.WriteCloser) error)
	realGCtxFuncWithReturn := fun.Function.(func(gctx.Context, io.WriteCloser) error)
	realFCtxFuncWithReturn := fun.Function.(func(context.CancelContext, io.WriteCloser) error)

	switch fun.Context {
	case internal.NoContext:
		if fun.Return == internal.NoReturn {
			realFunc(wopCloser{Writer: &outgoing})
		}

		if fun.Return == internal.ErrorReturn {
			err = realFuncWithReturn(wopCloser{Writer: &outgoing})
		}
	case internal.UseGoogleContext:
		err = execWithContext(func(ctx gctx.Context) error {
			if fun.Return == internal.NoReturn {
				realGCtxFunc(ctx, wopCloser{Writer: &outgoing})
			}

			if fun.Return == internal.ErrorReturn {
				err = realGCtxFuncWithReturn(ctx, wopCloser{Writer: &outgoing})
			}

			return nil
		}, 0)
	case internal.UseFauxCancelContext:
		err = execWithContext(func(ctx context.CancelContext) error {
			if fun.Return == internal.NoReturn {
				realFCtxFunc(ctx, wopCloser{Writer: &outgoing})
			}

			if fun.Return == internal.ErrorReturn {
				err = realFCtxFuncWithReturn(ctx, wopCloser{Writer: &outgoing})
			}

			return nil
		}, 0)
	}

	if err != nil {
		tests.Failed("Function %q with alias %q failed StringOnlyFunction criterias: %+q", fun.Name, fun.NS, err)
		return
	}

	tests.Passed("Function %q with alias %q passes StringOnlyFunction criterias", fun.Name, fun.NS)
}

// TestReaderFunction validates the behaviour of a function that expects a reader argument.
func TestReaderFunction(fun internal.ShogunFunc) {
	var err error

	defer func() {
		if rec := recover(); rec != nil {
			switch drec := rec.(type) {
			case error:
				err = drec
			default:
				err = fmt.Errorf("Recover Error: %+q", rec)
			}
		}
	}()

	var incoming bytes.Buffer
	incoming.WriteString(`{"name":"Rock"}`)

	realFunc := fun.Function.(func(io.Reader))
	realGCtxFunc := fun.Function.(func(gctx.Context, io.Reader))
	realFCtxFunc := fun.Function.(func(context.CancelContext, io.Reader))

	realFuncWithReturn := fun.Function.(func(io.Reader) error)
	realGCtxFuncWithReturn := fun.Function.(func(gctx.Context, io.Reader) error)
	realFCtxFuncWithReturn := fun.Function.(func(context.CancelContext, io.Reader) error)

	switch fun.Context {
	case internal.NoContext:
		if fun.Return == internal.NoReturn {
			realFunc(&incoming)
		}

		if fun.Return == internal.ErrorReturn {
			err = realFuncWithReturn(&incoming)
		}
	case internal.UseGoogleContext:
		err = execWithContext(func(ctx gctx.Context) error {
			if fun.Return == internal.NoReturn {
				realGCtxFunc(ctx, &incoming)
			}

			if fun.Return == internal.ErrorReturn {
				err = realGCtxFuncWithReturn(ctx, &incoming)
			}

			return nil
		}, 0)
	case internal.UseFauxCancelContext:
		err = execWithContext(func(ctx context.CancelContext) error {
			if fun.Return == internal.NoReturn {
				realFCtxFunc(ctx, &incoming)
			}

			if fun.Return == internal.ErrorReturn {
				err = realFCtxFuncWithReturn(ctx, &incoming)
			}

			return nil
		}, 0)
	}

	if err != nil {
		tests.Failed("Function %q with alias %q failed StringOnlyFunction criterias: %+q", fun.Name, fun.NS, err)
		return
	}

	tests.Passed("Function %q with alias %q passes StringOnlyFunction criterias", fun.Name, fun.NS)
	return
}

// TestReaderWithWriterFunction validates the behaviour of a function that expects a reader and WriteCloser argument.
func TestReaderWithWriterFunction(fun internal.ShogunFunc) {
	var err error

	defer func() {
		if rec := recover(); rec != nil {
			switch drec := rec.(type) {
			case error:
				err = drec
			default:
				err = fmt.Errorf("Recover Error: %+q", rec)
			}
		}
	}()

	var incoming, outgoing bytes.Buffer
	incoming.WriteString(`{"name":"Rock"}`)

	realFunc := fun.Function.(func(io.Reader, io.WriteCloser))
	realGCtxFunc := fun.Function.(func(gctx.Context, io.Reader, io.WriteCloser))
	realFCtxFunc := fun.Function.(func(context.CancelContext, io.Reader, io.WriteCloser))

	realFuncWithReturn := fun.Function.(func(io.Reader, io.WriteCloser) error)
	realGCtxFuncWithReturn := fun.Function.(func(gctx.Context, io.Reader, io.WriteCloser) error)
	realFCtxFuncWithReturn := fun.Function.(func(context.CancelContext, io.Reader, io.WriteCloser) error)

	switch fun.Context {
	case internal.NoContext:
		if fun.Return == internal.NoReturn {
			realFunc(&incoming, wopCloser{Writer: &outgoing})
		}

		if fun.Return == internal.ErrorReturn {
			err = realFuncWithReturn(&incoming, wopCloser{Writer: &outgoing})
		}
	case internal.UseGoogleContext:
		err = execWithContext(func(ctx gctx.Context) error {
			if fun.Return == internal.NoReturn {
				realGCtxFunc(ctx, &incoming, wopCloser{Writer: &outgoing})
			}

			if fun.Return == internal.ErrorReturn {
				err = realGCtxFuncWithReturn(ctx, &incoming, wopCloser{Writer: &outgoing})
			}

			return nil
		}, 0)
	case internal.UseFauxCancelContext:
		err = execWithContext(func(ctx context.CancelContext) error {
			if fun.Return == internal.NoReturn {
				realFCtxFunc(ctx, &incoming, wopCloser{Writer: &outgoing})
			}

			if fun.Return == internal.ErrorReturn {
				err = realFCtxFuncWithReturn(ctx, &incoming, wopCloser{Writer: &outgoing})
			}

			return nil
		}, 0)
	}

	if err != nil {
		tests.Failed("Function %q with alias %q failed StringOnlyFunction criterias: %+q", fun.Name, fun.NS, err)
		return
	}

	if outgoing.Len() == 0 {
		tests.Failed("Function %q with alias %q should have responded with output", fun.Name, fun.NS)
	}

	tests.Passed("Function %q with alias %q passes StringOnlyFunction criterias", fun.Name, fun.NS)
	return
}

// TestMapFunction validates the behaviour of a function that expects a map argument.
func TestMapFunction(fun internal.ShogunFunc) {
	var err error

	defer func() {
		if rec := recover(); rec != nil {
			switch drec := rec.(type) {
			case error:
				err = drec
			default:
				err = fmt.Errorf("Recover Error: %+q", rec)
			}
		}
	}()

	var incoming bytes.Buffer
	incoming.WriteString(`{"name":"Rock"}`)

	realFunc := fun.Function.(func(map[string]interface{}))
	realGCtxFunc := fun.Function.(func(gctx.Context, map[string]interface{}))
	realFCtxFunc := fun.Function.(func(context.CancelContext, map[string]interface{}))

	realFuncWithReturn := fun.Function.(func(map[string]interface{}) error)
	realGCtxFuncWithReturn := fun.Function.(func(gctx.Context, map[string]interface{}) error)
	realFCtxFuncWithReturn := fun.Function.(func(context.CancelContext, map[string]interface{}) error)

	data := make(map[string]interface{})
	if jserr := json.NewDecoder(&incoming).Decode(&data); jserr != nil {
		tests.Failed("Function %q with alias %q failed StringOnlyFunction criterias: %+q", fun.Name, fun.NS, jserr)
		return
	}

	switch fun.Context {
	case internal.NoContext:
		if fun.Return == internal.NoReturn {
			realFunc(data)
		}

		if fun.Return == internal.ErrorReturn {
			err = realFuncWithReturn(data)
		}
	case internal.UseGoogleContext:
		err = execWithContext(func(ctx gctx.Context) error {
			if fun.Return == internal.NoReturn {
				realGCtxFunc(ctx, data)
			}

			if fun.Return == internal.ErrorReturn {
				err = realGCtxFuncWithReturn(ctx, data)
			}

			return nil
		}, 0)
	case internal.UseFauxCancelContext:
		err = execWithContext(func(ctx context.CancelContext) error {
			if fun.Return == internal.NoReturn {
				realFCtxFunc(ctx, data)
			}

			if fun.Return == internal.ErrorReturn {
				err = realFCtxFuncWithReturn(ctx, data)
			}

			return nil
		}, 0)
	}

	if err != nil {
		tests.Failed("Function %q with alias %q failed StringOnlyFunction criterias: %+q", fun.Name, fun.NS, err)
		return
	}

	tests.Passed("Function %q with alias %q passes StringOnlyFunction criterias", fun.Name, fun.NS)
	return
}

// TestMapWithWriterFunction validates the behaviour of a function that expects a string argument.
func TestMapWithWriterFunction(fun internal.ShogunFunc) {
	var err error

	defer func() {
		if rec := recover(); rec != nil {
			switch drec := rec.(type) {
			case error:
				err = drec
			default:
				err = fmt.Errorf("Recover Error: %+q", rec)
			}
		}
	}()

	var incoming, outgoing bytes.Buffer
	incoming.WriteString(`{"name":"Rock"}`)

	realFunc := fun.Function.(func(map[string]interface{}, io.WriteCloser))
	realGCtxFunc := fun.Function.(func(gctx.Context, map[string]interface{}, io.WriteCloser))
	realFCtxFunc := fun.Function.(func(context.CancelContext, map[string]interface{}, io.WriteCloser))

	realFuncWithReturn := fun.Function.(func(map[string]interface{}, io.WriteCloser) error)
	realGCtxFuncWithReturn := fun.Function.(func(gctx.Context, map[string]interface{}, io.WriteCloser) error)
	realFCtxFuncWithReturn := fun.Function.(func(context.CancelContext, map[string]interface{}, io.WriteCloser) error)

	data := make(map[string]interface{})
	if jserr := json.NewDecoder(&incoming).Decode(&data); jserr != nil {
		tests.Failed("Function %q with alias %q failed StringOnlyFunction criterias: %+q", fun.Name, fun.NS, jserr)
		return
	}

	switch fun.Context {
	case internal.NoContext:
		if fun.Return == internal.NoReturn {
			realFunc(data, wopCloser{Writer: &outgoing})
		}

		if fun.Return == internal.ErrorReturn {
			err = realFuncWithReturn(data, wopCloser{Writer: &outgoing})
		}
	case internal.UseGoogleContext:
		err = execWithContext(func(ctx gctx.Context) error {
			if fun.Return == internal.NoReturn {
				realGCtxFunc(ctx, data, wopCloser{Writer: &outgoing})
			}

			if fun.Return == internal.ErrorReturn {
				err = realGCtxFuncWithReturn(ctx, data, wopCloser{Writer: &outgoing})
			}

			return nil
		}, 0)
	case internal.UseFauxCancelContext:
		err = execWithContext(func(ctx context.CancelContext) error {
			if fun.Return == internal.NoReturn {
				realFCtxFunc(ctx, data, wopCloser{Writer: &outgoing})
			}

			if fun.Return == internal.ErrorReturn {
				err = realFCtxFuncWithReturn(ctx, data, wopCloser{Writer: &outgoing})
			}

			return nil
		}, 0)
	}

	if err != nil {
		tests.Failed("Function %q with alias %q failed StringOnlyFunction criterias: %+q", fun.Name, fun.NS, err)
		return
	}

	if outgoing.Len() == 0 {
		tests.Failed("Function %q with alias %q should have responded with output", fun.Name, fun.NS)
	}

	tests.Passed("Function %q with alias %q passes StringOnlyFunction criterias", fun.Name, fun.NS)
	return
}

// TestNoArgumentFunction validates the behaviour of a function that expects no argument.
func TestNoArgumentFunction(fun internal.ShogunFunc) {
	var err error

	defer func() {
		if rec := recover(); rec != nil {
			switch drec := rec.(type) {
			case error:
				err = drec
			default:
				err = fmt.Errorf("Recover Error: %+q", rec)
			}
		}
	}()

	realFunc := fun.Function.(func())
	realGCtxFunc := fun.Function.(func(gctx.Context))
	realFCtxFunc := fun.Function.(func(context.CancelContext))

	realFuncWithReturn := fun.Function.(func() error)
	realGCtxFuncWithReturn := fun.Function.(func(gctx.Context) error)
	realFCtxFuncWithReturn := fun.Function.(func(context.CancelContext) error)

	switch fun.Context {
	case internal.NoContext:
		if fun.Return == internal.NoReturn {
			realFunc()
		}

		if fun.Return == internal.ErrorReturn {
			err = realFuncWithReturn()
		}
	case internal.UseGoogleContext:
		err = execWithContext(func(ctx gctx.Context) error {
			if fun.Return == internal.NoReturn {
				realGCtxFunc(ctx)
			}

			if fun.Return == internal.ErrorReturn {
				err = realGCtxFuncWithReturn(ctx)
			}

			return nil
		}, 0)
	case internal.UseFauxCancelContext:
		err = execWithContext(func(ctx context.CancelContext) error {
			if fun.Return == internal.NoReturn {
				realFCtxFunc(ctx)
			}

			if fun.Return == internal.ErrorReturn {
				err = realFCtxFuncWithReturn(ctx)
			}

			return nil
		}, 0)
	}

	if err != nil {
		tests.Failed("Function %q with alias %q failed StringOnlyFunction criterias: %+q", fun.Name, fun.NS, err)
		return
	}

	tests.Passed("Function %q with alias %q passes StringOnlyFunction criterias", fun.Name, fun.NS)
	return
}

// TestStringFunction validates the behaviour of a function that expects a string argument.
func TestStringFunction(fun internal.ShogunFunc) {
	var err error

	defer func() {
		if rec := recover(); rec != nil {
			switch drec := rec.(type) {
			case error:
				err = drec
			default:
				err = fmt.Errorf("Recover Error: %+q", rec)
			}
		}
	}()

	var incoming bytes.Buffer
	incoming.WriteString("Rock")

	realFunc := fun.Function.(func(string))
	realGCtxFunc := fun.Function.(func(gctx.Context, string))
	realFCtxFunc := fun.Function.(func(context.CancelContext, string))

	realFuncWithReturn := fun.Function.(func(string) error)
	realGCtxFuncWithReturn := fun.Function.(func(gctx.Context, string) error)
	realFCtxFuncWithReturn := fun.Function.(func(context.CancelContext, string) error)

	switch fun.Context {
	case internal.NoContext:
		if fun.Return == internal.NoReturn {
			realFunc(incoming.String())
		}

		if fun.Return == internal.ErrorReturn {
			err = realFuncWithReturn(incoming.String())
		}
	case internal.UseGoogleContext:
		err = execWithContext(func(ctx gctx.Context) error {
			if fun.Return == internal.NoReturn {
				realGCtxFunc(ctx, incoming.String())
			}

			if fun.Return == internal.ErrorReturn {
				err = realGCtxFuncWithReturn(ctx, incoming.String())
			}

			return nil
		}, 0)
	case internal.UseFauxCancelContext:
		err = execWithContext(func(ctx context.CancelContext) error {
			if fun.Return == internal.NoReturn {
				realFCtxFunc(ctx, incoming.String())
			}

			if fun.Return == internal.ErrorReturn {
				err = realFCtxFuncWithReturn(ctx, incoming.String())
			}

			return nil
		}, 0)
	}

	if err != nil {
		tests.Failed("Function %q with alias %q failed StringOnlyFunction criterias: %+q", fun.Name, fun.NS, err)
		return
	}

	tests.Passed("Function %q with alias %q passes StringOnlyFunction criterias", fun.Name, fun.NS)
	return
}

// TestStringWithWriterFunction validates the behaviour of a function that expects a string argument.
func TestStringWithWriterFunction(fun internal.ShogunFunc) {
	var err error

	defer func() {
		if rec := recover(); rec != nil {
			switch drec := rec.(type) {
			case error:
				err = drec
			default:
				err = fmt.Errorf("Recover Error: %+q", rec)
			}
		}
	}()

	var incoming, outgoing bytes.Buffer
	incoming.WriteString("Rock")

	realFunc := fun.Function.(func(string, io.WriteCloser))
	realGCtxFunc := fun.Function.(func(gctx.Context, string, io.WriteCloser))
	realFCtxFunc := fun.Function.(func(context.CancelContext, string, io.WriteCloser))

	realFuncWithReturn := fun.Function.(func(string, io.WriteCloser) error)
	realGCtxFuncWithReturn := fun.Function.(func(gctx.Context, string, io.WriteCloser) error)
	realFCtxFuncWithReturn := fun.Function.(func(context.CancelContext, string, io.WriteCloser) error)

	switch fun.Context {
	case internal.NoContext:
		if fun.Return == internal.NoReturn {
			realFunc(incoming.String(), wopCloser{Writer: &outgoing})
		}

		if fun.Return == internal.ErrorReturn {
			err = realFuncWithReturn(incoming.String(), wopCloser{Writer: &outgoing})
		}
	case internal.UseGoogleContext:
		err = execWithContext(func(ctx gctx.Context) error {
			if fun.Return == internal.NoReturn {
				realGCtxFunc(ctx, incoming.String(), wopCloser{Writer: &outgoing})
			}

			if fun.Return == internal.ErrorReturn {
				err = realGCtxFuncWithReturn(ctx, incoming.String(), wopCloser{Writer: &outgoing})
			}

			return nil
		}, 0)
	case internal.UseFauxCancelContext:
		err = execWithContext(func(ctx context.CancelContext) error {
			if fun.Return == internal.NoReturn {
				realFCtxFunc(ctx, incoming.String(), wopCloser{Writer: &outgoing})
			}

			if fun.Return == internal.ErrorReturn {
				err = realFCtxFuncWithReturn(ctx, incoming.String(), wopCloser{Writer: &outgoing})
			}

			return nil
		}, 0)
	}

	if err != nil {
		tests.Failed("Function %q with alias %q failed StringOnlyFunction criterias: %+q", fun.Name, fun.NS, err)
		return
	}

	if outgoing.Len() == 0 {
		tests.Failed("Function %q with alias %q should have responded with output", fun.Name, fun.NS)
	}

	tests.Passed("Function %q with alias %q passes StringOnlyFunction criterias", fun.Name, fun.NS)
	return
}

func execWithContext(fun interface{}, ctxTimeout time.Duration) error {
	switch dfunc := fun.(type) {
	case func(context.CancelContext) error:
		var ctx context.CancelableContext

		if ctxTimeout == 0 {
			ctx = context.NewCnclContext()
		} else {
			ctx = context.NewExpiringCnclContext(nil, ctxTimeout)
		}

		defer ctx.Cancel()

		return dfunc(ctx)
	case func(gctx.Context) error:
		var ctx gctx.Context
		var canceller func()

		if ctxTimeout == 0 {
			ctx = gctx.Background()
		} else {
			ctx, canceller = gctx.WithTimeout(gctx.Background(), ctxTimeout)
		}

		if canceller != nil {
			defer canceller()
		}

		return dfunc(ctx)
	}

	return errors.New("Unknown context type")
}

type wopCloser struct {
	io.Writer
}

// Close does nothing.
func (wopCloser) Close() error {
	return nil
}
