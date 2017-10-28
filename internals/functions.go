package internals

import (
	"os"
	"strings"
	"text/template"
)

const (
	spaceLen = 7
)

// FlagType defines a int type represent the type of flag a function wants.
type FlagType int

// const of flag types.
const (
	BadFlag FlagType = iota
	IntFlag
	Int64Flag
	UintFlag
	Uint64Flag
	StringFlag
	BoolFlag
	TBoolFlag
	DurationFlag
	Float64Flag
	IntSliceFlag
	Int64SliceFlag
	StringSliceFlag
	BoolSliceFlag
	AnyTypeFlag
	Float64SliceFlag
)

// GetFlag returns a FlagType for the giving name.
func GetFlag(name string) FlagType {
	switch name {
	case "Float64":
		return Float64Flag
	case "Duration":
		return DurationFlag
	case "TBool":
		return TBoolFlag
	case "Bool":
		return BoolFlag
	case "String":
		return StringFlag
	case "Uint":
		return UintFlag
	case "Uint64":
		return Uint64Flag
	case "Int":
		return IntFlag
	case "Int64":
		return Int64Flag
	case "IntSlice":
		return IntSliceFlag
	case "Int64Slice":
		return Int64SliceFlag
	case "BoolSlice":
		return BoolSliceFlag
	case "Float64Slice":
		return Float64SliceFlag
	case "StringSlice":
		return StringSliceFlag
	}

	return BadFlag
}

// Int returns the Flag type back as it's int value.
func (f FlagType) Int() int {
	return int(f)
}

// FlagType returns associated type.
func (f FlagType) String() string {
	switch f {
	case Float64Flag:
		return "Float64"
	case DurationFlag:
		return "Duration"
	case TBoolFlag:
		return "TBool"
	case BoolFlag:
		return "Bool"
	case StringFlag:
		return "String"
	case UintFlag:
		return "Uint"
	case Uint64Flag:
		return "Uint64"
	case IntFlag:
		return "Int"
	case Int64Flag:
		return "Int64"
	case IntSliceFlag:
		return "IntSlice"
	case Int64SliceFlag:
		return "Int64Slice"
	case BoolSliceFlag:
		return "BoolSlice"
	case Float64SliceFlag:
		return "Float64Slice"
	case StringSliceFlag:
		return "StringSlice"
	}

	return "Unknown"
}

// ReturnType defines a int type represent the type of return a function provides.
type ReturnType int

// Int returns the real int value of the typr.
func (f ReturnType) Int() int {
	return int(f)
}

// const for return state.
const (
	NoReturn ReturnType = iota + 1
	ErrorReturn
	UnknownErrorReturn
)

// ExportType defines a int type represent the export state of a function.
type ExportType int

// Int returns the real int value of the typr.
func (f ExportType) Int() int {
	return int(f)
}

// const for type export state.
const (
	UnExportedImport ExportType = iota + 1
	ExportedImport
)

// ContextType defines a int type represent the type of context argument a function receives.
type ContextType int

// Int returns the real int value of the typr.
func (f ContextType) Int() int {
	return int(f)
}

// consts for use or absence of context.
const (
	NoContext ContextType = iota + 1
	UseGoogleContext
	UseFauxContext
	UseUnknownContext
)

// ArgType defines a int type represent the type of arguments a function receives.
type ArgType int

// Int returns the real int value of the typr.
func (f ArgType) Int() int {
	return int(f)
}

// const for input state.
const (
	NoArgument                                    ArgType = iota + 1 // is func()
	WithContextArgument                                              // is func(Context)
	WithStringArgument                                               // is func(string)
	WithStringSliceArgument                                          // is func(string)
	WithMapArgument                                                  // is func(map[string]interface{})
	WithStructArgument                                               // is func(Movie)
	WithImportedObjectArgument                                       // is func(types.IMovie)
	WithReaderArgument                                               // is func(io.Reader)
	WithWriteCloserArgument                                          // is func(io.WriteCloser)
	WithStringArgumentAndWriteCloserArgument                         // is func(string, io.WriteCloser)
	WithStringSliceArgumentAndWriteCloserArgument                    // is func(string, io.WriteCloser)
	WithStructAndWriteCloserArgument                                 // is func(Movie, io.WriteCloser)
	WithMapAndWriteCloserArgument                                    // is func(map[string]interface{}, io.WriteCloser)
	WithImportedAndWriteCloserArgument                               // is func(types.IMovie, io.WriteCloser)
	WithReaderAndWriteCloserArgument                                 // is func(io.Reader, io.WriteCloser)
	WithUnknownArgument
)

var (
	// ArgumentFunctions contains functions to validate type.
	ArgumentFunctions = template.FuncMap{
		"returnsError": func(d ReturnType) bool {
			return d == ErrorReturn
		},
		"usesNoContext": func(d ContextType) bool {
			return d == NoContext
		},
		"usesGoogleContext": func(d ContextType) bool {
			return d == UseGoogleContext
		},
		"usesFauxContext": func(d ContextType) bool {
			return d == UseFauxContext
		},
		"hasArgumentStructExported": func(d ExportType) bool {
			return d == ExportedImport
		},
		"hasArgumentStructUnexported": func(d ExportType) bool {
			return d == UnExportedImport
		},
		"hasNoArgument": func(d ArgType) bool {
			return d == NoArgument
		},
		"hasContextArgument": func(d ArgType) bool {
			return d == WithContextArgument
		},
		"hasStringSliceArgument": func(d ArgType) bool {
			return d == WithStringSliceArgument
		},
		"hasStringArgument": func(d ArgType) bool {
			return d == WithStringArgument
		},
		"hasMapArgument": func(d ArgType) bool {
			return d == WithMapArgument
		},
		"hasStructArgument": func(d ArgType) bool {
			return d == WithStructArgument
		},
		"hasReadArgument": func(d ArgType) bool {
			return d == WithReaderArgument
		},
		"hasWriteArgument": func(d ArgType) bool {
			return d == WithWriteCloserArgument
		},
		"hasImportedArgument": func(d ArgType) bool {
			return d == WithImportedObjectArgument
		},
		"hasStringSliceArgumentWithWriter": func(d ArgType) bool {
			return d == WithStringSliceArgumentAndWriteCloserArgument
		},
		"hasStringArgumentWithWriter": func(d ArgType) bool {
			return d == WithStringArgumentAndWriteCloserArgument
		},
		"hasReadArgumentWithWriter": func(d ArgType) bool {
			return d == WithReaderAndWriteCloserArgument
		},
		"hasStructArgumentWithWriter": func(d ArgType) bool {
			return d == WithStructAndWriteCloserArgument
		},
		"hasMapArgumentWithWriter": func(d ArgType) bool {
			return d == WithMapAndWriteCloserArgument
		},
		"hasImportedArgumentWithWriter": func(d ArgType) bool {
			return d == WithImportedAndWriteCloserArgument
		},
	}
)

// ShogunFunc defines a type which contains a function definition details.
type ShogunFunc struct {
	NS       string      `json:"ns"`
	Type     ArgType     `json:"type"`
	Return   ReturnType  `json:"return"`
	Context  ContextType `json:"context"`
	Name     string      `json:"name"`
	Source   string      `json:"source"`
	Flags    Flags       `json:"flags"`
	Function interface{} `json:"-"`
}

// Flag contains details related to a provided flag.
type Flag struct {
	EnvVar string
	Name   string
	Desc   string
	Type   FlagType
}

// UsesEnv returns true/false if the flags can use an environment variable name.
func (f Flag) UsesEnv() bool {
	return f.EnvVar != ""
}

// FromList attempts to pull giving Flag value from list.
func (f Flag) FromList(args []string) (string, bool) {
	for _, arg := range args {
		vals := strings.Split(arg, "=")
		if len(vals) == 0 {
			continue
		}

		name := vals[0]
		if arg != name {
			continue
		}

		if f.Type == BoolFlag {
			return "true", true
		}

		if f.Type == TBoolFlag {
			return "false", true
		}

		if len(vals) > 1 {
			return vals[1], true
		}
	}

	return "", false
}

// FromEnv attempts to pull giving Flag value from environment.
func (f Flag) FromEnv() (string, bool) {
	return os.LookupEnv(f.EnvVar)
}

// VarMeta defines a struct to hold object details.
type VarMeta struct {
	Import     string
	ImportNick string
	Type       string
	TypeAddr   string
	Exported   ExportType
}

// Function defines a struct type that represent meta details of a giving function.
type Function struct {
	Context               ContextType
	Type                  ArgType
	Return                ReturnType
	StructExported        ExportType
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
	Flags                 Flags
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
		if item.Context == UseFauxContext {
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
