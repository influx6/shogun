package hexwriter

import "io"

var lowerhex = "0123456789abcdef"

// HexWriter transforms the writes the incoming data in hex format.
type HexWriter struct {
	io.Writer
}

// New returns a new instance of a HexWriter.
func New(w io.Writer) *HexWriter {
	return &HexWriter{Writer: w}
}

// Write meets the io.Write interface Write method
func (hx HexWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	var sod = []byte(`\x00`)
	var b byte

	for n, b = range p {
		sod[2] = lowerhex[b/16]
		sod[3] = lowerhex[b%16]
		hx.Writer.Write(sod)
	}

	n++

	return
}
