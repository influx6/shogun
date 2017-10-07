package internal

// const defines available ExecType for shogun selected types
const (
	NoValue ExecType = iota
	NoInErrReturn
	ReaderInErrReturn
	ReaderInNoErrReturn
	CancelContextInErrReturn
	CancelContextInNoErrReturn
)

// type NoValueFunc func()

// type NoInErrReturnValueFunc func() error

// type ContextInNoValueFunc func(CancelContext)

// type ContextInErrReturnValueFunc func(CancelContext) error

// type MapInReturnErrorFunc func(CancelContext, map[string]interface{}) error

// type StructInReturnErrorFunc func(CancelContext, Movie{Name string `json:"name"`}) error

// type TypeInReturnErrorFunc func(CancelContext, interface{}) error

// type TypeWriteCloserInReturnErrorFunc func(CancelContext, interface{}, io.WriteCloser) error

// type ReaderInReturnErrorFunc func(CancelContext, io.Reader) error

// type ReaderWriteCloserInReturnErrorFunc func(CancelContext, io.Reader, io.WriteCloser) error

// ExecType defines a hint type to represent a giving function argument and return type.
type ExecType int

// CancelContext defines a type which provides Done signal for cancelling operations.
type CancelContext interface {
	Done() <-chan struct{}
}

// Function defines a struct type that represent meta details of a giving function.
type Function struct {
	UseContext  bool
	Type        ExecType
	Name        string
	Exported    bool
	From        string
	Synopses    string
	Source      string
	Description string
	Depends     []string
}
