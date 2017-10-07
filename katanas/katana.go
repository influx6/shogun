// +build shogun

// Package katanas provides exported functions as tasks runnable from commandline.
//
// @binaryName(name => shogun-shell)
//
package katanas

import (
	"io"

	"github.com/influx6/shogun/internal"
)

type Woofer struct {
	Name   string `json:"name"`
	Caller string `json:"caller"`
}

func Draw() {}

// Slash is the default tasks due to below annotation.
// @default
func Slash() error {
	return nil
}

func Buba(ctx internal.CancelContext) {
}

func Bob(ctx internal.CancelContext) error {
	return nil
}

func Jija(ctx internal.CancelContext, mp Woofer) error {
	return nil
}

func Juga(ctx internal.CancelContext, r io.Reader) error {
	return nil
}

func Buba(ctx internal.CancelContext, mp interface{}) error {
	return nil
}

func Biga(ctx internal.CancelContext, r io.Reader, w io.WriteCloser) error {
	return nil
}

func Bub(ctx internal.CancelContext, mp map[string]interface{}) error {
	return nil
}

func Guga(ctx internal.CancelContext, mp interface{}, w io.WriteCloser) error {
	return nil
}
