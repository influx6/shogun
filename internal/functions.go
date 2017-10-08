package internal

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
	Context     int
	Type        int
	Return      int
	Name        string
	Exported    bool
	From        string
	Synopses    string
	Source      string
	Description string
	Depends     []string
	Imports     []VarMeta
}
