package internal

import (
	"strings"
)

const (
	spaceLen = 7
)

// const for return state.
const (
	NoReturn = iota << 10
	ErrorReturn
	UnknownErrorReturn
)

// consts for use or absence of context.
const (
	NoContext = iota << 30
	UseGoogleContext
	UseFauxCancelContext
	UseValueBagContext
	UseUnknownContext
)

// const for input state.
const (
	NoArgument                          = iota << 20 // is func()
	WithMapArgument                                  // is func(map[string]interface{})
	WithStructArgument                               // is func(Movie)
	WithInterfaceArgument                            // is func(IMovie)
	WithImportedObjectArgument                       // is func(types.IMovie)
	WithReaderArgument                               // is func(io.Reader)
	WithWriteCloserArgument                          // is func(io.WriteCloser)
	WithStructAndWriteCloserArgument                 // is func(Movie, io.WriteCloser)
	WithInterfaceAndWriteCloserArgument              // is func(IMovie, io.WriteCloser)
	WithImportedAndWriteCloserArgument               // is func(types.IMovie, io.WriteCloser)
	WithReaderAndWriteCloserArgument                 // is func(io.Reader, io.WriteCloser)
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
	Context         int
	Type            int
	Return          int
	Exported        bool
	Name            string
	From            string
	Synopses        string
	Source          string
	Description     string
	Package         string
	PackagePath     string
	PackageFile     string
	PackageFileName string
	Depends         []string
	Imports         []VarMeta
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
