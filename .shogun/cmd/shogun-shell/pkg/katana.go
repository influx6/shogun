package pkg

import (
	"io"

	"github.com/influx6/faux/context"
	ty "github.com/influx6/shogun/katanas/types"
)

type wondra struct {
	Name string
}

func Draw() {}

// Slash is the default tasks due to below annotation.
// @default
func Slash() error {
	return nil
}

// Buba is bub.
func Buba(ctx context.ValueBagContext) {
}

func Bob(ctx context.CancelContext) error {
	return nil
}

func Jija(ctx context.CancelContext, mp ty.Woofer) error {
	return nil
}

func JijaPointer(ctx context.CancelContext, mp *ty.Woofer) error {
	return nil
}

func Juga(ctx context.CancelContext, r io.Reader) error {
	return nil
}

func Boba(ctx context.CancelContext, mp ty.IBlob) error {
	return nil
}

func Biga(ctx context.CancelContext, r io.Reader, w io.WriteCloser) error {
	return nil
}

func Nack(ctx context.CancelContext, mp map[string]interface{}) error {
	return nil
}

func Rulla(ctx context.CancelContext, mp wondra, w io.WriteCloser) error {
	return nil
}

func Hulla(ctx context.CancelContext, mp *wondra, w io.WriteCloser) error {
	return nil
}

func Guga(ctx context.CancelContext, mp ty.IBlob, w io.WriteCloser) error {
	return nil
}
