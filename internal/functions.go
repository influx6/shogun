package internal

import (
	"strings"
)

const (
	spaceLen = 7
)

// const for return state.
const (
	NoReturn = iota + 1
	ErrorReturn
	UnknownErrorReturn
)

// consts for use or absence of context.
const (
	NoContext = iota + 5
	UseGoogleContext
	UseFauxCancelContext
	UseUnknownContext
)

// const for input state.
const (
	NoArgument                         = iota + 10 // is func()
	WithContextArgument                            // is func(Context)
	WithMapArgument                                // is func(map[string]interface{})
	WithStructArgument                             // is func(Movie)
	WithImportedObjectArgument                     // is func(types.IMovie)
	WithReaderArgument                             // is func(io.Reader)
	WithWriteCloserArgument                        // is func(io.WriteCloser)
	WithStructAndWriteCloserArgument               // is func(Movie, io.WriteCloser)
	WithMapAndWriteCloserArgument                  // is func(map[string]interface{}, io.WriteCloser)
	WithImportedAndWriteCloserArgument             // is func(types.IMovie, io.WriteCloser)
	WithReaderAndWriteCloserArgument               // is func(io.Reader, io.WriteCloser)
	WithUnknownArgument
)

// VarMeta defines a struct to hold object details.
type VarMeta struct {
	Import     string
	ImportNick string
	Type       string
	TypeAddr   string
}

// Function defines a struct type that represent meta details of a giving function.
type Function struct {
	Context               int
	Type                  int
	Return                int
	Exported              bool
	Default               bool
	RealName              string
	Name                  string
	From                  string
	Synopses              string
	Source                string
	Description           string
	Package               string
	PackagePath           string
	PackageFile           string
	PackageFileName       string
	HelpMessage           string
	HelpMessageWithSource string
	Depends               []string
	Imports               VarMeta
	ContextImport         VarMeta
}

// PackageFunctions holds a package level function with it's path and name.
type PackageFunctions struct {
	Name       string
	Hash       string
	Path       string
	Desc       string
	FilePath   string
	BinaryName string
	MaxNameLen int
	List       []Function
}

// Default returns the function set has default for when the execution is called.
func (pn PackageFunctions) Default() []Function {
	for _, item := range pn.List {
		if item.Default {
			return []Function{item}
		}
	}

	return nil
}

// HasFauxImports returns true/false if any part of the function uses faux context.
func (pn PackageFunctions) HasFauxImports() bool {
	for _, item := range pn.List {
		if item.Context == UseFauxCancelContext {
			return true
		}
	}

	return false
}

// HasGoogleImports returns true/false if any part of the function uses google context.
func (pn PackageFunctions) HasGoogleImports() bool {
	for _, item := range pn.List {
		if item.Context == UseGoogleContext {
			return true
		}
	}

	return false
}

// Imports returns a map of all import paths for giving package functions.
func (pn PackageFunctions) Imports() map[string]string {
	mo := make(map[string]string)

	for _, item := range pn.List {
		if item.Imports.Import == "" {
			continue
		}

		if _, ok := mo[item.Imports.Import]; !ok {
			mo[item.Imports.Import] = item.Imports.ImportNick
		}
	}

	return mo
}

// SpaceFor returns space value for a giving name.
func (pn PackageFunctions) SpaceFor(name string) string {
	nmLength := len(name)

	if nmLength == pn.MaxNameLen {
		return printSpaceLine(spaceLen)
	}

	if nmLength < pn.MaxNameLen {
		diff := pn.MaxNameLen - nmLength
		return printSpaceLine(spaceLen + diff)
	}

	newLen := spaceLen - (pn.MaxNameLen - nmLength)
	if newLen < -1 {
		newLen *= -1
	}

	return printSpaceLine(newLen)
}

func printSpaceLine(length int) string {
	var lines []string

	for i := 0; i < length; i++ {
		lines = append(lines, " ")
	}

	return strings.Join(lines, "")
}
