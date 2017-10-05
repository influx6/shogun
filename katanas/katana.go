// +build shogun

// Package katanas provides exported functions as tasks runnable from commandline.
//
// @binaryName(name => shogun-shell)
//
package katanas


// Slash is the default tasks due to below annotation.
// @default
func Slash() error {
  return nil
}
