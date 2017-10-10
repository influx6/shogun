package types

// IBlob ...
type IBlob interface {
	Run() error
}

// Woofer ...
type Woofer struct {
	Name   string `json:"name"`
	Caller string `json:"caller"`
}
