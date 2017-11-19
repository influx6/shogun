package gen

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	tm "text/template"

	"github.com/influx6/moz/gen/templates"
)

//go:generate go generate ./templates/...

//======================================================================================================================

var (
	// CommaWriter defines the a writer that consistently writes a ','.
	CommaWriter = NewConstantWriter([]byte(","))

	// NewlineWriter defines the a writer that consistently writes a \n.
	NewlineWriter = NewConstantWriter([]byte("\n"))

	// CommaSpacedWriter defines the a writer that consistently writes a ', '.
	CommaSpacedWriter = NewConstantWriter([]byte(", "))

	// PeriodWriter defines the a writer that consistently writes a '.'.
	PeriodWriter = NewConstantWriter([]byte("."))
)

//======================================================================================================================

// WriteDirective defines a struct which contains giving directives as to the file and
// the relative path within which it should be written to.
// Include are tags which give meta description of the optionality of each field.
type WriteDirective struct {
	Writer       io.WriterTo `ast:"-,!optional"`       // WriteTo which contains the complete content of the file to be written to.
	Dir          string      `ast:"dir,optional"`      // Relative dir path written into it if not existing.
	FileName     string      `ast:"filename,optional"` // alternative fileName to use for new file.
	DontOverride bool        `ast:"dont_override,optional"`
	Before       func() error
	After        func() error
}

//======================================================================================================================

// WriterToMap defines a int64erface which maps giving declaration values
// int64o appropriate form for final output. It allows us create custom wrappers to
// define specific output style for a giving set of declarations.
type WriterToMap interface {
	Map(...io.WriterTo) io.WriterTo
}

// MapOut defines an function type which maps giving
// data retrieved from a series of readers int64o the provided byte slice, returning
// the total number of data written and any error encountered.
type MapOut func(io.Writer, ...io.WriterTo) (int64, error)

//======================================================================================================================

// WritersTo defines the body contents of a giving declaration/structure.
type WritersTo []io.WriterTo

// WriteTo writes to the provided writer the variable declaration.
func (d WritersTo) WriteTo(w io.Writer) (int64, error) {
	wc := NewWriteCounter(w)

	for _, item := range d {
		if _, err := item.WriteTo(wc); IsNotDrainError(err) {
			return wc.Written(), err
		}
	}

	return wc.Written(), nil
}

// Map applies a giving declaration mapper to the underlying io.Readers of the io.WriterTo.
func (d WritersTo) Map(mp WriterToMap) io.WriterTo {
	return mp.Map(d...)
}

//======================================================================================================================

// MapAnyWriter applies a giving set of MapOut functions with the provided int64ernal declarations
// writes to the provided io.Writer.
type MapAnyWriter struct {
	Map MapOut
	Dcl []io.WriterTo
}

// WriteTo takes the data slice and writes int64ernal WritersTo int64o the giving writer.
func (m MapAnyWriter) WriteTo(to io.Writer) (int64, error) {
	return m.Map(to, m.Dcl...)
}

//======================================================================================================================

// MapAny defines a struct which implements a structure which uses the provided
// int64ernal MapOut function to apply the necessary business logic of copying
// giving data space by a giving series of readers.
type MapAny struct {
	MapFn MapOut
}

// Map takes a giving set of readers returning a structure which implements the io.Reader int64erface
// for copying underlying data to the expected output.
func (mapper MapAny) Map(dls ...io.WriterTo) io.WriterTo {
	return MapAnyWriter{Map: mapper.MapFn, Dcl: dls}
}

//======================================================================================================================

// NewlineMapper defines a struct which implements the DeclarationMap which maps a set of
// items by seperating their output with a period '.', but execludes before the first and
// after the last item.
var NewlineMapper = MapAny{MapFn: func(to io.Writer, declrs ...io.WriterTo) (int64, error) {
	wc := NewWriteCounter(to)

	total := len(declrs) - 1

	for index, declr := range declrs {
		if _, err := declr.WriteTo(wc); err != nil && err != io.EOF {
			return 0, err
		}

		if index < total {
			NewlineWriter.WriteTo(wc)
		}
	}

	return wc.Written(), nil
}}

// DotMapper defines a struct which implements the DeclarationMap which maps a set of
// items by seperating their output with a period '.', but execludes before the first and
// after the last item.
var DotMapper = MapAny{MapFn: func(to io.Writer, declrs ...io.WriterTo) (int64, error) {
	wc := NewWriteCounter(to)

	total := len(declrs) - 1

	for index, declr := range declrs {
		if _, err := declr.WriteTo(wc); err != nil && err != io.EOF {
			return 0, err
		}

		if index < total {
			PeriodWriter.WriteTo(wc)
		}
	}

	return wc.Written(), nil
}}

// CommaSpacedMapper defines a struct which implements the DeclarationMap which maps a set of
// items by seperating their output with a coma ', ', but execludes before the first and
// after the last item.
var CommaSpacedMapper = MapAny{MapFn: func(to io.Writer, declrs ...io.WriterTo) (int64, error) {
	wc := NewWriteCounter(to)

	total := len(declrs) - 1

	for index, declr := range declrs {
		if _, err := declr.WriteTo(wc); err != nil && err != io.EOF {
			return 0, err
		}

		if index < total {
			CommaSpacedWriter.WriteTo(wc)
		}
	}

	return wc.Written(), nil
}}

// CommaMapper defines a struct which implements the DeclarationMap which maps a set of
// items by seperating their output with a coma ',', but execludes before the first and
// after the last item.
var CommaMapper = MapAny{MapFn: func(to io.Writer, declrs ...io.WriterTo) (int64, error) {
	wc := NewWriteCounter(to)

	total := len(declrs) - 1

	for index, declr := range declrs {
		if _, err := declr.WriteTo(wc); err != nil && err != io.EOF {
			return 0, err
		}

		if index < total {
			CommaWriter.WriteTo(wc)
		}
	}

	return wc.Written(), nil
}}

//======================================================================================================================

var (
	defaultMapType = bytes.NewBuffer([]byte("map"))
)

// MapDeclr defines a type for a map declaration.
type MapDeclr struct {
	Type    NameDeclr
	Value   NameDeclr
	MapType io.WriterTo
	Values  map[string]io.WriterTo
}

// WriteTo writes to the provided writer the variable declaration.
func (tx MapDeclr) WriteTo(w io.Writer) (int64, error) {
	wc := NewWriteCounter(NewNoBOM(w))

	tml, err := ToTemplate("mapDeclr", templates.Must("map.tml"), nil)
	if err != nil {
		return 0, err
	}

	values := make(map[string]string)

	var b bytes.Buffer

	for name, item := range tx.Values {
		b.Reset()

		if _, err := item.WriteTo(&b); IsNotDrainError(err) {
			return 0, err
		}

		values[name] = b.String()
	}

	mapType := io.WriterTo(defaultMapType)

	if tx.MapType != nil {
		mapType = tx.MapType
	}

	var mapTypeName bytes.Buffer

	if _, err := mapType.WriteTo(&mapTypeName); IsNotDrainError(err) {
		return 0, err
	}

	var bind = struct {
		MapType string
		Type    string
		Value   string
		Values  map[string]string
	}{
		Values:  values,
		Type:    tx.Type.Name,
		Value:   tx.Value.Name,
		MapType: mapTypeName.String(),
	}

	if err := tml.Execute(wc, bind); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// TextDeclr defines a declaration type which takes a giving source text and generate text.Template for it
// and providing binding and will execute the template to generate it's output
type TextDeclr struct {
	Template string
	Name     string
	Binding  interface{}
	Funcs    tm.FuncMap
}

// WriteTo writes to the provided writer the text declaration.
func (tx TextDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := tm.New(tx.Name).Funcs(defaultFuncs).Funcs(tx.Funcs).Parse(tx.Template)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, tx.Binding); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// SourceDeclr defines a declaration type which takes a giving source template
// and providing binding and will execute the template to generate it's output
type SourceDeclr struct {
	Template *tm.Template
	Binding  interface{}
}

// WriteTo writes to the provided writer the variable declaration.
func (src SourceDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	wc := NewWriteCounter(w)

	if err := src.Template.Execute(wc, src.Binding); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// PackageDeclr defines a declaration which generates a go package source.
type PackageDeclr struct {
	Name io.WriterTo `json:"name"`
	Body WritersTo   `json:"body"`
}

// WriteTo writes to the provided writer the variable declaration.
func (pkg PackageDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("packageDeclr", templates.Must("package.tml"), nil)
	if err != nil {
		return 0, err
	}

	var named, body bytes.Buffer

	if _, err := pkg.Name.WriteTo(&named); IsNotDrainError(err) {
		return 0, err
	}

	if _, err := pkg.Body.WriteTo(&body); IsNotDrainError(err) {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Name string
		Body string
	}{
		Name: named.String(),
		Body: body.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// TypeDeclr defines a declaration struct for representing a giving type.
type TypeDeclr struct {
	TypeName string `json:"typeName"`
}

// String returns the int64ernal name associated with the TypeDeclr.
func (t TypeDeclr) String() string {
	return t.TypeName
}

// WriteTo writes to the provided writer the variable declaration.
func (t TypeDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("typeDeclr", templates.Must("variable-type-only.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Type string
	}{
		Type: t.TypeName,
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// NameDeclr defines a declaration struct for representing a giving value.
type NameDeclr struct {
	Name string `json:"name"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n NameDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("nameDeclr", templates.Must("name.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Name string
	}{
		Name: n.Name,
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal name associated with the NameDeclr.
func (n NameDeclr) String() string {
	return n.Name
}

//======================================================================================================================

// RuneASCIIDeclr defines a declaration struct for representing a giving value.
type RuneASCIIDeclr struct {
	Value rune `json:"value"`
}

// String returns the internal data associated with the structure.
func (n RuneASCIIDeclr) String() string {
	return strconv.QuoteRuneToASCII(n.Value)
}

// WriteTo writes to the provided writer the variable declaration.
func (n RuneASCIIDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("runeASCIIDeclr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// RuneGraphicsDeclr defines a declaration struct for representing a giving value.
type RuneGraphicsDeclr struct {
	Value rune `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n RuneGraphicsDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("runeGraphicsDeclr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n RuneGraphicsDeclr) String() string {
	return strconv.QuoteRuneToGraphic(n.Value)
}

// RuneDeclr defines a declaration struct for representing a giving value.
type RuneDeclr struct {
	Value rune `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n RuneDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("runeDeclr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n RuneDeclr) String() string {
	return strconv.QuoteRune(n.Value)
}

// StringASCIIDeclr defines a declaration struct for representing a giving value.
type StringASCIIDeclr struct {
	Value string `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n StringASCIIDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("stringASCIIDeclr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n StringASCIIDeclr) String() string {
	return strconv.QuoteToASCII(n.Value)
}

// StringDeclr defines a declaration struct for representing a giving value.
type StringDeclr struct {
	Value string `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n StringDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("stringDeclr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n StringDeclr) String() string {
	return strconv.Quote(n.Value)
}

// BoolDeclr defines a declaration struct for representing a giving value.
type BoolDeclr struct {
	Value bool `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n BoolDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("boolDeclr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n BoolDeclr) String() string {
	return strconv.FormatBool(n.Value)
}

// UIntBaseDeclr defines a declaration struct for representing a giving value.
type UIntBaseDeclr struct {
	Value uint64 `json:"value"`
	Base  int    `json:"base"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n UIntBaseDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("uintBaseDeclr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n UIntBaseDeclr) String() string {
	return strconv.FormatUint(n.Value, n.Base)
}

// UInt64Declr defines a declaration struct for representing a giving value.
type UInt64Declr struct {
	Value uint64 `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n UInt64Declr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("uint64Declr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n UInt64Declr) String() string {
	return strconv.FormatUint(n.Value, 10)
}

// UInt32Declr defines a declaration struct for representing a giving value.
type UInt32Declr struct {
	Value uint32 `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n UInt32Declr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("uint32Declr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n UInt32Declr) String() string {
	return strconv.FormatUint(uint64(n.Value), 10)
}

// IntBaseDeclr defines a declaration struct for representing a giving value.
type IntBaseDeclr struct {
	Value int64 `json:"value"`
	Base  int   `json:"base"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n IntBaseDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("intBaseDeclr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n IntBaseDeclr) String() string {
	return strconv.FormatInt(n.Value, n.Base)
}

// Int64Declr defines a declaration struct for representing a giving value.
type Int64Declr struct {
	Value int64 `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n Int64Declr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("int64Declr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n Int64Declr) String() string {
	return strconv.FormatInt(n.Value, 10)
}

// Int32Declr defines a declaration struct for representing a giving value.
type Int32Declr struct {
	Value int32 `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n Int32Declr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("int32Declr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n Int32Declr) String() string {
	return strconv.FormatInt(int64(n.Value), 10)
}

// IntDeclr defines a declaration struct for representing a giving value.
type IntDeclr struct {
	Value int `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n IntDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("intDeclr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n IntDeclr) String() string {
	return strconv.Itoa(n.Value)
}

// FloatBaseDeclr defines a declaration struct for representing a giving value.
type FloatBaseDeclr struct {
	Value     float64 `json:"value"`
	Bitsize   int     `json:"base"`
	Precision int     `json:"precision"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n FloatBaseDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("floatBaseDeclr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n FloatBaseDeclr) String() string {
	return strconv.FormatFloat(n.Value, 'f', n.Precision, n.Bitsize)
}

// Float32Declr defines a declaration struct for representing a giving value.
type Float32Declr struct {
	Value float32 `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n Float32Declr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("float32Declr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n Float32Declr) String() string {
	return strconv.FormatFloat(float64(n.Value), 'f', 4, 32)
}

// Float64Declr defines a declaration struct for representing a giving value.
type Float64Declr struct {
	Value float64 `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n Float64Declr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("float64Declr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n Float64Declr) String() string {
	return strconv.FormatFloat(n.Value, 'f', 4, 64)
}

// ValueDeclr defines a declaration struct for representing a giving value.
type ValueDeclr struct {
	Value          interface{}              `json:"value"`
	ValueConverter func(interface{}) string `json:"-"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n ValueDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("valueDeclr", templates.Must("value.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Value string
	}{
		Value: n.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal data associated with the structure.
func (n ValueDeclr) String() string {
	return n.ValueConverter(n.Value)
}

//======================================================================================================================

// SliceTypeDeclr defines a declaration struct for representing a go slice.
type SliceTypeDeclr struct {
	Type TypeDeclr `json:"type"`
}

// WriteTo writes to the provided writer the variable declaration.
func (t SliceTypeDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("sliceTypeDeclr", templates.Must("slicetype.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, t); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// SliceDeclr defines a declaration struct for representing a go slice.
type SliceDeclr struct {
	Type   TypeDeclr     `json:"type"`
	Values []io.WriterTo `json:"values"`
}

// WriteTo writes to the provided writer the variable declaration.
func (t SliceDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	var vam bytes.Buffer

	if _, err := CommaMapper.Map(t.Values...).WriteTo(&vam); err != nil && err != io.EOF {
		return 0, err
	}

	tml, err := ToTemplate("sliceDeclr", templates.Must("slicevalue.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Type   string
		Values string
	}{
		Type:   t.Type.String(),
		Values: vam.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// Contains different sets of operator declarations.
var (
	PlusOperator           = OperatorDeclr{Operation: "+"}
	MinusOperator          = OperatorDeclr{Operation: "-"}
	ModeOperator           = OperatorDeclr{Operation: "%"}
	DivideOperator         = OperatorDeclr{Operation: "/"}
	MultiplicationOperator = OperatorDeclr{Operation: "*"}
	EqualOperator          = OperatorDeclr{Operation: "=="}
	LessThanOperator       = OperatorDeclr{Operation: "<"}
	MoreThanOperator       = OperatorDeclr{Operation: ">"}
	LessThanEqualOperator  = OperatorDeclr{Operation: "<="}
	MoreThanEqualOperator  = OperatorDeclr{Operation: ">="}
	NotEqualOperator       = OperatorDeclr{Operation: "!="}
	ANDOperator            = OperatorDeclr{Operation: "&&"}
	OROperator             = OperatorDeclr{Operation: "||"}
	BinaryANDOperator      = OperatorDeclr{Operation: "&"}
	BinaryOROperator       = OperatorDeclr{Operation: "|"}
	DecrementOperator      = OperatorDeclr{Operation: "--"}
	IncrementOperator      = OperatorDeclr{Operation: "++"}
)

// OperatorDeclr defines a declaration which produces a variable declaration.
type OperatorDeclr struct {
	Operation string `json:"operation"`
}

// String returns the internal name associated with the struct.
func (n OperatorDeclr) String() string {
	return n.Operation
}

// WriteTo writes the giving representation into the provided writer.
func (n OperatorDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	total, err := w.Write([]byte(n.Operation))
	return int64(total), err
}

//======================================================================================================================

// VariableTypeDeclr defines a declaration which produces a variable declaration.
type VariableTypeDeclr struct {
	Name NameDeclr `json:"name"`
	Type TypeDeclr `json:"typename"`
}

// WriteTo writes to the provided writer the variable declaration.
func (v VariableTypeDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("variableDeclr", templates.Must("var-variable-type.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, v); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// FieldTypeDeclr defines a declaration which produces a variable declaration.
type FieldTypeDeclr struct {
	Name NameDeclr `json:"name"`
	Type TypeDeclr `json:"typename"`
}

// WriteTo writes to the provided writer the variable declaration.
func (v FieldTypeDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("variableDeclr", templates.Must("variable-type.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, v); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// VariableNameDeclr defines a declaration which produces a variable declaration.
type VariableNameDeclr struct {
	Name NameDeclr `json:"name"`
}

// WriteTo writes to the provided writer the variable declaration.
func (v VariableNameDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("variableDeclr", templates.Must("variable-name.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, v); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// VariableAssignmentDeclr defines a declaration which produces a variable declaration.
type VariableAssignmentDeclr struct {
	Name  NameDeclr   `json:"name"`
	Value io.WriterTo `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (v VariableAssignmentDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("variableDeclr", templates.Must("variable-assign-basic.tml"), nil)
	if err != nil {
		return 0, err
	}

	var vam bytes.Buffer

	if _, err := v.Value.WriteTo(&vam); err != nil && err != io.EOF {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Name  string
		Value string
	}{
		Name:  v.Name.String(),
		Value: vam.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// VariableShortAssignmentDeclr defines a declaration which produces a variable declaration.
type VariableShortAssignmentDeclr struct {
	Name  NameDeclr   `json:"name"`
	Value io.WriterTo `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (v VariableShortAssignmentDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("variableDeclr", templates.Must("variable-assign.tml"), nil)
	if err != nil {
		return 0, err
	}

	var vam bytes.Buffer

	if _, err := v.Value.WriteTo(&vam); err != nil && err != io.EOF {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Name  string
		Value string
	}{
		Name:  v.Name.String(),
		Value: vam.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// ValueAssignmentDeclr defines a declaration which produces a variable declaration.
type ValueAssignmentDeclr struct {
	Name  NameDeclr   `json:"name"`
	Value io.WriterTo `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (v ValueAssignmentDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	tml, err := ToTemplate("valueAssignmentDeclr", templates.Must("value-assign.tml"), nil)
	if err != nil {
		return 0, err
	}

	var vam bytes.Buffer

	if _, err := v.Value.WriteTo(&vam); err != nil && err != io.EOF {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Name  string
		Value string
	}{
		Name:  v.Name.String(),
		Value: vam.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// SuffixDeclr defines a declaration which produces a block with the provided prefix.
type SuffixDeclr struct {
	Suffix io.WriterTo `json:"-"`
	Value  io.WriterTo `json:"-"`
}

// WriteTo writes the giving representation into the provided writer.
func (b SuffixDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	wc := NewWriteCounter(w)

	if _, err := b.Value.WriteTo(wc); err != nil && err != io.EOF {
		return 0, err
	}

	if b.Suffix != nil {
		if _, err := b.Suffix.WriteTo(wc); IsNotDrainError(err) {
			return 0, err
		}
	}

	return wc.Written(), nil
}

// PrefixDeclr defines a declaration which produces a block with the provided prefix.
type PrefixDeclr struct {
	Prefix io.WriterTo `json:"-"`
	Value  io.WriterTo `json:"-"`
}

// WriteTo writes the giving representation into the provided writer.
func (b PrefixDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	wc := NewWriteCounter(w)

	if b.Prefix != nil {
		if _, err := b.Prefix.WriteTo(wc); IsNotDrainError(err) {
			return 0, err
		}
	}

	if _, err := b.Value.WriteTo(wc); err != nil && err != io.EOF {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// SingleByteBlockDeclr defines a declaration which produces a block byte slice which is written to a writer.
// declaration writer into it's block char.
// eg. A BlockDeclr with Char '{{'
// 		Will produce '{{DataFROMWriter' output.
type SingleByteBlockDeclr struct {
	Block []byte `json:"block"`
}

// WriteTo writes the giving representation into the provided writer.
func (b SingleByteBlockDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	wc := NewWriteCounter(w)

	if _, err := wc.Write(b.Block); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// SingleBlockDeclr defines a declaration which produces a block char which is written to a writer.
// eg. A BlockDeclr with Char '{'
// 		Will produce '{' output.
type SingleBlockDeclr struct {
	Rune rune `json:"rune"`
}

// WriteTo writes the giving representation into the provided writer.
func (b SingleBlockDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	wc := NewWriteCounter(w)

	if _, err := wc.Write([]byte{byte(b.Rune)}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// ByteBlockDeclr defines a declaration which produces a block cover which wraps any other
// declaration writer into it's block char.
// eg. A BlockDeclr with Char '{''}'
// 		Will produce '{{DataFROMWriter}}' output.
type ByteBlockDeclr struct {
	Block      io.WriterTo `json:"block"`
	BlockBegin []byte      `json:"begin"`
	BlockEnd   []byte      `json:"end"`
}

// WriteTo writes the giving representation into the provided writer.
func (b ByteBlockDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)
	wc := NewWriteCounter(w)

	if b.BlockBegin != nil {
		if _, err := wc.Write(b.BlockBegin); err != nil {
			return 0, err
		}
	}

	if _, err := b.Block.WriteTo(wc); err != nil && err != io.EOF {
		return 0, err
	}

	if b.BlockEnd != nil {
		if _, err := wc.Write(b.BlockEnd); err != nil {
			return 0, err
		}
	}

	return wc.Written(), nil
}

// BlockDeclr defines a declaration which produces a block cover which wraps any other
// declaration writer into it's block char.
// eg. A BlockDeclr with Char '{''}'
// 		Will produce '{DataFROMWriter}' output.
type BlockDeclr struct {
	Block     io.WriterTo `json:"block"`
	RuneBegin rune        `json:"begin"`
	RuneEnd   rune        `json:"end"`
}

// WriteTo writes the giving representation into the provided writer.
func (b BlockDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	wc := NewWriteCounter(w)

	if _, err := wc.Write([]byte{byte(b.RuneBegin)}); err != nil {
		return 0, err
	}

	if _, err := b.Block.WriteTo(wc); err != nil && err != io.EOF {
		return 0, err
	}

	if _, err := wc.Write([]byte{byte(b.RuneEnd)}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// JSONBlock defines a block area which is encycled by braces.
// {
// 	...
// }
type JSONBlock struct {
	Items map[string]io.WriterTo
}

// WriteTo writes to the provided writer the variable declaration.
func (v JSONBlock) WriteTo(w io.Writer) (int64, error) {
	wc := NewWriteCounter(NewNoBOM(w))

	tml, err := ToTemplate("jsonBlockDeclr", templates.Must("jsonblock.tml"), nil)
	if err != nil {
		return 0, err
	}

	var content []string

	var bu bytes.Buffer
	for name, item := range v.Items {
		bu.Reset()

		if _, err := item.WriteTo(&bu); IsNotDrainError(err) {
			return 0, err
		}

		content = append(content, fmt.Sprintf("%q:	%s", name, strings.Join(strings.Split(bu.String(), "\n"), "\n\t")))
	}

	if err := tml.Execute(wc, content); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// JSONDeclr defines a block area contains other JSONBlocks.
// {
// 	  {
//		...
// 	   }
// }
type JSONDeclr struct {
	Documents []io.WriterTo
}

// WriteTo writes to the provided writer the variable declaration.
func (m JSONDeclr) WriteTo(w io.Writer) (int64, error) {
	wc := NewWriteCounter(NewNoBOM(w))

	tml, err := ToTemplate("jsonDeclr", templates.Must("json.tml"), nil)
	if err != nil {
		return 0, err
	}

	var contents []string

	var bu bytes.Buffer

	for _, doc := range m.Documents {
		bu.Reset()

		if _, err := doc.WriteTo(&bu); IsNotDrainError(err) {
			return 0, err
		}

		contents = append(contents, bu.String())
	}

	if err := tml.Execute(wc, struct {
		Documents []string
	}{
		Documents: contents,
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// ConditionDeclr defines a declaration which produces a variable declaration.
type ConditionDeclr struct {
	PreVar   VariableNameDeclr `json:"prevar"`
	PostVar  VariableNameDeclr `json:"postvar"`
	Operator OperatorDeclr     `json:"operator"`
}

// WriteTo writes the giving representation into the provided writer.
func (c ConditionDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	wc := NewWriteCounter(w)

	if _, err := c.PreVar.WriteTo(wc); err != nil && err != io.EOF {
		return 0, err
	}

	if _, err := c.Operator.WriteTo(wc); err != nil && err != io.EOF {
		return 0, err
	}

	if _, err := c.PostVar.WriteTo(wc); err != nil && err != io.EOF {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// FunctionDeclr defines a declaration which produces function about based on the giving
// constructor and body.
type FunctionDeclr struct {
	Name        NameDeclr        `json:"name"`
	Constructor ConstructorDeclr `json:"constructor"`
	Returns     io.WriterTo      `json:"returns"`
	Body        WritersTo        `json:"body"`
}

// WriteTo writes to the provided writer the function declaration.
func (f FunctionDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	var constr, returns, body bytes.Buffer

	if _, err := f.Constructor.WriteTo(&constr); IsNotDrainError(err) {
		return 0, err
	}

	if _, err := f.Returns.WriteTo(&returns); IsNotDrainError(err) {
		return 0, err
	}

	if _, err := f.Body.WriteTo(&body); IsNotDrainError(err) {
		return 0, err
	}

	var declr = struct {
		Name        string
		Returns     string
		Body        string
		Constructor string
	}{
		Name:        f.Name.String(),
		Returns:     returns.String(),
		Body:        body.String(),
		Constructor: constr.String(),
	}

	tml, err := ToTemplate("functionDeclr", templates.Must("function.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, declr); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// FunctionTypeDeclr defines a declaration which produces function about based on the giving
// constructor and body.
type FunctionTypeDeclr struct {
	Name        NameDeclr        `json:"name"`
	Constructor ConstructorDeclr `json:"constructor"`
	Returns     io.WriterTo      `json:"returns"`
}

// WriteTo writes to the provided writer the function declaration.
func (f FunctionTypeDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	var constr, returns bytes.Buffer

	if _, err := f.Constructor.WriteTo(&constr); IsNotDrainError(err) {
		return 0, err
	}

	if _, err := f.Returns.WriteTo(&returns); IsNotDrainError(err) {
		return 0, err
	}

	var declr = struct {
		Name        string
		Returns     string
		Constructor string
	}{
		Name:        f.Name.String(),
		Returns:     returns.String(),
		Constructor: constr.String(),
	}

	tml, err := ToTemplate("functionTypeDeclr", templates.Must("function-type.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, declr); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// TagDeclr defines a declaration for representing go type tags.
type TagDeclr struct {
	Format string `json:"format"`
	Name   string `json:"name"`
}

// WriteTo writes to the provided writer the variable declaration.
func (v TagDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	tml, err := ToTemplate("tagDeclr", templates.Must("tag.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, v); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// StructTypeDeclr defines a declaration which produces a variable declaration.
type StructTypeDeclr struct {
	Name NameDeclr `json:"name"`
	Type TypeDeclr `json:"typename"`
	Tags WritersTo `json:"tags"`
}

// WriteTo writes to the provided writer the variable declaration.
func (v StructTypeDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	tml, err := ToTemplate("structTypeDeclr", templates.Must("structtype.tml"), nil)
	if err != nil {
		return 0, err
	}

	var tags bytes.Buffer
	tags.WriteRune('`')
	if _, err := v.Tags.WriteTo(&tags); IsNotDrainError(err) {
		return 0, err
	}
	tags.WriteRune('`')

	wc := NewWriteCounter(w)
	if err := tml.Execute(wc, struct {
		Name string
		Type string
		Tags string
	}{
		Name: v.Name.String(),
		Type: v.Type.String(),
		Tags: tags.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// StructDeclr defines a declaration struct for representing a single comment.
type StructDeclr struct {
	Name        NameDeclr   `json:"name"`
	Type        TypeDeclr   `json:"type"`
	Comments    io.WriterTo `json:"comments"`
	Annotations io.WriterTo `json:"annotations"`
	Fields      WritersTo   `json:"fields"`
}

// WriteTo writes to the provided writer the variable declaration.
func (v StructDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	tml, err := ToTemplate("structDeclr", templates.Must("struct.tml"), nil)
	if err != nil {
		return 0, err
	}

	var fields []string
	var comments, annotations bytes.Buffer

	if _, err := v.Comments.WriteTo(&comments); IsNotDrainError(err) {
		return 0, err
	}

	if _, err := v.Annotations.WriteTo(&annotations); IsNotDrainError(err) {
		return 0, err
	}

	var b bytes.Buffer
	for _, item := range v.Fields {
		b.Reset()

		if _, err := item.WriteTo(&b); IsNotDrainError(err) {
			return 0, err
		}

		fields = append(fields, b.String())
	}

	wc := NewWriteCounter(w)
	if err := tml.Execute(wc, struct {
		Name        string
		Type        string
		Comments    string
		Annotations string
		Fields      []string
	}{
		Fields:      fields,
		Name:        v.Name.String(),
		Type:        v.Type.String(),
		Comments:    comments.String(),
		Annotations: annotations.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// CommentDeclr defines a declaration struct for representing a single comment.
type CommentDeclr struct {
	MainBlock io.WriterTo `json:"mainBlock"`
	Blocks    WritersTo   `json:"blocks"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n CommentDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	tml, err := ToTemplate("commentDeclr", templates.Must("comments.tml"), nil)
	if err != nil {
		return 0, err
	}

	var mainBlock bytes.Buffer
	var blocks []string

	if _, err := n.MainBlock.WriteTo(&mainBlock); IsNotDrainError(err) {
		return 0, err
	}

	var bu bytes.Buffer
	for _, block := range n.Blocks {
		bu.Reset()

		if _, err := block.WriteTo(&bu); IsNotDrainError(err) {
			return 0, err
		}

		blocks = append(blocks, strings.Split(bu.String(), "\n")...)
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		MainBlock string
		Blocks    []string
	}{
		MainBlock: mainBlock.String(),
		Blocks:    blocks,
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// MultiCommentDeclr defines a declaration struct for representing a single comment.
type MultiCommentDeclr struct {
	MainBlock io.WriterTo `json:"mainBlock"`
	Blocks    WritersTo   `json:"blocks"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n MultiCommentDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	tml, err := ToTemplate("multiCommentDeclr", templates.Must("multicomments.tml"), nil)
	if err != nil {
		return 0, err
	}

	var mainBlock bytes.Buffer
	var blocks []string

	if _, err := n.MainBlock.WriteTo(&mainBlock); IsNotDrainError(err) {
		return 0, err
	}

	var bu bytes.Buffer
	for _, block := range n.Blocks {
		bu.Reset()

		if _, err := block.WriteTo(&bu); IsNotDrainError(err) {
			return 0, err
		}

		blocks = append(blocks, strings.Split(bu.String(), "\n")...)
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		MainBlock string
		Blocks    []string
	}{
		MainBlock: mainBlock.String(),
		Blocks:    blocks,
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// AnnotationDeclr defines a struct for generating a annotation.
type AnnotationDeclr struct {
	Value string `json:"value"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n AnnotationDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	tml, err := ToTemplate("annotationDeclr", templates.Must("annotations.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, n); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal name associated with the NameDeclr.
func (n AnnotationDeclr) String() string {
	return n.Value
}

//======================================================================================================================

// TextBlockDeclr defines a declaration struct for representing a single comment.
type TextBlockDeclr struct {
	Text string `json:"text"`
}

// WriteTo writes to the provided writer the variable declaration.
func (n TextBlockDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	wc := NewWriteCounter(w)

	if _, err := wc.Write([]byte(n.Text)); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// String returns the internal name associated with the NameDeclr.
func (n TextBlockDeclr) String() string {
	return n.Text
}

//======================================================================================================================

// CustomReturnDeclr defines a declaration which produces argument based output
// of it's giving internals.
type CustomReturnDeclr struct {
	Returns WritersTo `json:"returns"`
}

// WriteTo writes to the provided writer the function argument declaration.
func (f CustomReturnDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	arguments := CommaMapper.Map(f.Returns...)

	return (BlockDeclr{
		Block:     arguments,
		RuneBegin: '(',
		RuneEnd:   ')',
	}).WriteTo(w)
}

// ReturnDeclr defines a declaration which produces argument based output
// of it's giving internals.
type ReturnDeclr struct {
	Returns []TypeDeclr `json:"returns"`
}

// WriteTo writes to the provided writer the function argument declaration.
func (f ReturnDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	if len(f.Returns) == 0 {
		return 0, nil
	}

	var decals []io.WriterTo

	for _, item := range f.Returns {
		decals = append(decals, item)
	}

	arguments := CommaMapper.Map(decals...)

	return (BlockDeclr{
		Block:     arguments,
		RuneBegin: '(',
		RuneEnd:   ')',
	}).WriteTo(w)
}

//======================================================================================================================

// ConstructorDeclr defines a declaration which produces argument based output
// of it's giving internals.
type ConstructorDeclr struct {
	Arguments []VariableTypeDeclr `json:"constructor"`
}

// WriteTo writes to the provided writer the function argument declaration.
func (f ConstructorDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	var decals []io.WriterTo

	for _, item := range f.Arguments {
		decals = append(decals, item)
	}

	arguments := CommaSpacedMapper.Map(decals...)

	return (BlockDeclr{
		Block:     arguments,
		RuneBegin: '(',
		RuneEnd:   ')',
	}).WriteTo(w)
}

//======================================================================================================================

// ImportItemDeclr defines a type to represent a import statement.
type ImportItemDeclr struct {
	Path      string `json:"path"`
	Namespace string `json:"namespace"`
}

// WriteTo writes to the provided writer the structure declaration.
func (im ImportItemDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	tml, err := ToTemplate("importItemDeclr", templates.Must("import-item.tml"), nil)
	if err != nil {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, &im); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// ImportDeclr defines a type to represent a import statement.
type ImportDeclr struct {
	Packages []ImportItemDeclr `json:"packages"`
}

// WriteTo writes to the provided writer the structure declaration.
func (im ImportDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	tml, err := ToTemplate("importDeclr", templates.Must("import.tml"), nil)
	if err != nil {
		return 0, err
	}

	var pkgs []string

	var b bytes.Buffer
	for _, item := range im.Packages {
		b.Reset()

		if _, err := item.WriteTo(&b); IsNotDrainError(err) {
			return 0, err
		}

		pkgs = append(pkgs, b.String())
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Imports []string
	}{
		Imports: pkgs,
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// IfDeclr defines a type to represent a if condition.
type IfDeclr struct {
	Condition io.WriterTo
	Action    io.WriterTo
}

// WriteTo writes to the provided writer the structure declaration.
func (c IfDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	tml, err := ToTemplate("ifDeclr", templates.Must("if.tml"), nil)
	if err != nil {
		return 0, err
	}

	block := ByteBlockDeclr{
		Block:      c.Condition,
		BlockBegin: []byte("("),
		BlockEnd:   []byte(")"),
	}

	var action, condition bytes.Buffer

	if _, err := block.WriteTo(&condition); IsNotDrainError(err) {
		return 0, err
	}

	if _, err := c.Action.WriteTo(&action); IsNotDrainError(err) {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Condition string
		Action    string
	}{
		Condition: condition.String(),
		Action:    action.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// DefaultCaseDeclr defines a structure which generates switch default case declarations.
type DefaultCaseDeclr struct {
	Behaviour io.WriterTo
}

// WriteTo writes to the provided writer the structure declaration.
func (c DefaultCaseDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	tml, err := ToTemplate("defaultCaseDeclr", templates.Must("case-default.tml"), nil)
	if err != nil {
		return 0, err
	}

	var caseAction bytes.Buffer

	if _, err := c.Behaviour.WriteTo(&caseAction); IsNotDrainError(err) {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Action string
	}{
		Action: caseAction.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// CaseDeclr defines a structure which generates switch case declarations.
type CaseDeclr struct {
	Condition io.WriterTo
	Behaviour io.WriterTo
}

// WriteTo writes to the provided writer the structure declaration.
func (c CaseDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	tml, err := ToTemplate("caseDeclr", templates.Must("case.tml"), nil)
	if err != nil {
		return 0, err
	}

	var caseCondition, caseAction bytes.Buffer

	if _, err := c.Condition.WriteTo(&caseCondition); IsNotDrainError(err) {
		return 0, err
	}

	if _, err := c.Behaviour.WriteTo(&caseAction); IsNotDrainError(err) {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Action    string
		Condition string
	}{
		Action:    caseAction.String(),
		Condition: caseCondition.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

// SwitchDeclr defines a structure which generates switch declarations.
type SwitchDeclr struct {
	Condition io.WriterTo
	Cases     []CaseDeclr
	Default   DefaultCaseDeclr
}

// WriteTo writes to the provided writer the structure declaration.
func (c SwitchDeclr) WriteTo(w io.Writer) (int64, error) {
	w = NewNoBOM(w)

	tml, err := ToTemplate("caseDeclr", templates.Must("case.tml"), nil)
	if err != nil {
		return 0, err
	}

	var caseCondition, caseDefault, caseAction bytes.Buffer

	for _, item := range c.Cases {
		if _, err := item.WriteTo(&caseAction); IsNotDrainError(err) {
			return 0, err
		}
	}

	if c.Default.Behaviour != nil {
		if _, err := c.Default.WriteTo(&caseDefault); IsNotDrainError(err) {
			return 0, err
		}
	}

	if _, err := c.Condition.WriteTo(&caseCondition); IsNotDrainError(err) {
		return 0, err
	}

	wc := NewWriteCounter(w)

	if err := tml.Execute(wc, struct {
		Condition string
		Cases     string
		Default   string
	}{
		Cases:     caseAction.String(),
		Default:   caseDefault.String(),
		Condition: caseCondition.String(),
	}); err != nil {
		return 0, err
	}

	return wc.Written(), nil
}

//======================================================================================================================

// NoBOMWriter removes any unwanted characters like \x00 found in
// possible code.
type NoBOMWriter struct {
	io.Writer
}

// NewNoBOM returns a new instance of the NoBOMWriter
func NewNoBOM(w io.Writer) *NoBOMWriter {
	return &NoBOMWriter{
		Writer: w,
	}
}

// Write writes the bytes provided after removing such any \x00
// characters.
func (bom *NoBOMWriter) Write(b []byte) (int, error) {
	var bm []byte

	for _, bl := range b {
		if bl == byte('\x00') {
			continue
		}

		bm = append(bm, bl)
	}

	return bom.Writer.Write(bm)
}

//======================================================================================================================

func cleanSpace(ns ...string) []string {
	for index, item := range ns {
		ns[index] = strings.TrimSpace(item)
	}

	return ns
}
