package templates

//go:generate go run generate.go

import (
	"fmt"
)

//go:generate go run generate.go
var files = make(map[string][]byte)

// Must attempts to retrieve the file data if found else panics.
func Must(file string) []byte {
	data, err := Get(file)
	if err != nil {
		panic(err)
	}

	return data
}

// Get retrieves the giving file data from the map store if it exists.
func Get(file string) ([]byte, error) {
	data, ok := files[file]
	if !ok {
		return nil, fmt.Errorf("File data for %q not found", file)
	}

	return data, nil
}

func init() {

}
