package gen

import (
	"bufio"
	"io"
)

var (
	dataSize = 512 * 1024 // 512kb
)

// FromReader implements io.WriterTo by wrapping a provided io.Reader.
type FromReader struct {
	R             io.Reader
	ReadBlockSize int
}

// WriteTo implements io.WriterTo.
func (fm *FromReader) WriteTo(w io.Writer) (int64, error) {
	if fm.ReadBlockSize <= 0 {
		fm.ReadBlockSize = dataSize
	}

	buf := bufio.NewReader(fm.R)

	data := make([]byte, fm.ReadBlockSize)

	var totalWritten int64

	for {
		read, err := buf.Read(data)
		if err != nil && err != io.EOF {
			return totalWritten, err
		}

		if err != nil && err == io.EOF {
			return totalWritten, nil
		}

		if read >= dataSize {
			written, err2 := w.Write(data[:read])
			if err2 != nil {
				return totalWritten, err2
			}

			totalWritten += int64(written)
			continue
		}

		written, err := w.Write(data[:read])
		if err != nil {
			return totalWritten, err
		}

		totalWritten += int64(written)
	}
}

//======================================================================================================================

// WriteCounter defines a struct which collects write counts of
// a giving io.Writer
type WriteCounter struct {
	io.Writer
	written int64
}

// NewWriteCounter returns a new instance of the WriteCounter.
func NewWriteCounter(w io.Writer) *WriteCounter {
	return &WriteCounter{Writer: w}
}

// Written returns the total number of data writer to the underline writer.
func (w *WriteCounter) Written() int64 {
	return w.written
}

// Write calls the internal io.Writer.Write method and adds up
// the write counts.
func (w *WriteCounter) Write(data []byte) (int, error) {
	inc, err := w.Writer.Write(data)

	w.written += int64(inc)

	return inc, err
}

//======================================================================================================================

// IsNoError returns true/false if the error is nil.
func IsNoError(err error) bool {
	return err == nil
}

// IsDrainError is used to check if a error value matches io.EOF.
func IsDrainError(err error) bool {
	if err != nil && err == io.EOF {
		return true
	}

	return false
}

// IsNotDrainError is used to check if a error value matches io.EOF.
func IsNotDrainError(err error) bool {
	if err != nil && err != io.EOF {
		return true
	}

	return false
}

//======================================================================================================================

// ConstantWriter defines a writer that consistently writes a provided output.
type ConstantWriter struct {
	d []byte
}

// NewConstantWriter returns a new instance of ConstantWriter.
func NewConstantWriter(d []byte) ConstantWriter {
	return ConstantWriter{d: d}
}

// WriteTo writes the data provided into the writer.
func (cw ConstantWriter) WriteTo(w io.Writer) (int64, error) {
	total, err := w.Write(cw.d)
	return int64(total), err
}
