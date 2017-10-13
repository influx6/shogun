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

	files["shogun-add.tml"] = []byte("\x2f\x2f\x20\x2b\x62\x75\x69\x6c\x64\x20\x73\x68\x6f\x67\x75\x6e\x0a\x0a\x70\x61\x63\x6b\x61\x67\x65\x20\x7b\x7b\x20\x6c\x6f\x77\x65\x72\x20\x2e\x50\x61\x63\x6b\x61\x67\x65\x7d\x7d\x0a")
	files["shogun-in-pkg.tml"] = []byte("\x2f\x2f\x20\x2b\x62\x75\x69\x6c\x64\x20\x73\x68\x6f\x67\x75\x6e\x0a\x0a\x2f\x2f\x20\x50\x61\x63\x6b\x61\x67\x65\x20\x7b\x7b\x20\x6c\x6f\x77\x65\x72\x20\x2e\x50\x61\x63\x6b\x61\x67\x65\x20\x7d\x7d\x20\x70\x72\x6f\x76\x69\x64\x65\x73\x20\x65\x78\x70\x6f\x72\x74\x65\x64\x20\x66\x75\x6e\x63\x74\x69\x6f\x6e\x73\x20\x61\x73\x20\x74\x61\x73\x6b\x73\x20\x72\x75\x6e\x6e\x61\x62\x6c\x65\x20\x66\x72\x6f\x6d\x20\x63\x6f\x6d\x6d\x61\x6e\x64\x6c\x69\x6e\x65\x2e\x0a\x2f\x2f\x0a\x2f\x2f\x20\x40\x62\x69\x6e\x61\x72\x79\x4e\x61\x6d\x65\x28\x6e\x61\x6d\x65\x20\x3d\x3e\x20\x7b\x7b\x6c\x6f\x77\x65\x72\x20\x2e\x42\x69\x6e\x61\x72\x79\x4e\x61\x6d\x65\x20\x7d\x7d\x29\x0a\x2f\x2f\x0a\x70\x61\x63\x6b\x61\x67\x65\x20\x7b\x7b\x6c\x6f\x77\x65\x72\x20\x2e\x50\x61\x63\x6b\x61\x67\x65\x7d\x7d\x0a\x0a\x0a\x2f\x2f\x20\x53\x6c\x61\x73\x68\x20\x69\x73\x20\x74\x68\x65\x20\x64\x65\x66\x61\x75\x6c\x74\x20\x74\x61\x73\x6b\x73\x20\x64\x75\x65\x20\x74\x6f\x20\x62\x65\x6c\x6f\x77\x20\x61\x6e\x6e\x6f\x74\x61\x74\x69\x6f\x6e\x2e\x0a\x2f\x2f\x20\x40\x64\x65\x66\x61\x75\x6c\x74\x0a\x66\x75\x6e\x63\x20\x53\x6c\x61\x73\x68\x28\x29\x20\x65\x72\x72\x6f\x72\x20\x7b\x0a\x20\x20\x72\x65\x74\x75\x72\x6e\x20\x6e\x69\x6c\x0a\x7d\x0a")
	files["shogun-pkg-inbin-list.tml"] = []byte("\x53\x68\x6f\x67\x75\x6e\x20\x63\x6c\x61\x6e\x20\x7b\x7b\x20\x71\x75\x6f\x74\x65\x20\x2e\x4d\x61\x69\x6e\x2e\x46\x72\x6f\x6d\x50\x61\x63\x6b\x61\x67\x65\x7d\x7d\x0a\x0a\xe2\xa1\xbf\x20\x53\x41\x4d\x55\x52\x41\x49\x20\x7b\x7b\x20\x71\x75\x6f\x74\x65\x20\x2e\x4d\x61\x69\x6e\x2e\x42\x69\x6e\x61\x72\x79\x4e\x61\x6d\x65\x20\x7d\x7d\x20\x20\x20\x20\x20\x20\x20\x20\x7b\x7b\x2e\x4d\x61\x69\x6e\x2e\x44\x65\x73\x63\x7d\x7d\x0a\x0a\x4b\x41\x54\x41\x4e\x41\x20\x43\x4f\x4d\x4d\x41\x4e\x44\x53\x3a\x0a\x7b\x7b\x20\x72\x61\x6e\x67\x65\x20\x24\x69\x6e\x64\x65\x78\x2c\x20\x24\x65\x6c\x65\x6d\x20\x3a\x3d\x20\x2e\x4d\x61\x69\x6e\x2e\x46\x75\x6e\x63\x74\x69\x6f\x6e\x73\x7d\x7d\x7b\x7b\x20\x72\x61\x6e\x67\x65\x20\x24\x65\x6c\x65\x6d\x2e\x4c\x69\x73\x74\x7d\x7d\x0a\xe2\xa0\x99\x20\x7b\x7b\x20\x2e\x4e\x61\x6d\x65\x20\x7d\x7d\x7b\x7b\x24\x65\x6c\x65\x6d\x2e\x53\x70\x61\x63\x65\x46\x6f\x72\x20\x2e\x4e\x61\x6d\x65\x7d\x7d\x7b\x7b\x2e\x53\x79\x6e\x6f\x70\x73\x65\x73\x7d\x7d\x0a\x7b\x7b\x20\x65\x6e\x64\x20\x7d\x7d\x20\x7b\x7b\x20\x65\x6e\x64\x20\x7d\x7d\x0a\x0a\x7b\x7b\x20\x69\x66\x20\x6e\x6f\x74\x65\x71\x75\x61\x6c\x20\x28\x6c\x65\x6e\x20\x2e\x53\x75\x62\x73\x29\x20\x30\x7d\x7d\x4f\x54\x48\x45\x52\x20\x53\x41\x4d\x55\x52\x41\x49\x20\x43\x4f\x4d\x4d\x41\x4e\x44\x53\x3a\x0a\x7b\x7b\x20\x72\x61\x6e\x67\x65\x20\x24\x5f\x2c\x20\x24\x65\x6c\x65\x6d\x20\x3a\x3d\x20\x2e\x53\x75\x62\x73\x7d\x7d\x0a\xe2\xa1\xbf\x20\x7b\x7b\x20\x24\x65\x6c\x65\x6d\x2e\x42\x69\x6e\x61\x72\x79\x4e\x61\x6d\x65\x20\x7d\x7d\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x7b\x7b\x24\x65\x6c\x65\x6d\x2e\x44\x65\x73\x63\x7d\x7d\x0a\x7b\x7b\x20\x65\x6e\x64\x20\x7d\x7d\x7b\x7b\x65\x6e\x64\x7d\x7d\x0a\x48\x45\x4c\x50\x3a\x0a\x0a\x54\x6f\x20\x73\x65\x65\x20\x6d\x6f\x72\x65\x20\x6f\x6e\x20\x65\x61\x63\x68\x20\x63\x6f\x6d\x6d\x61\x6e\x64\x3a\x0a\x0a\x20\x20\x68\x65\x6c\x70\x20\x5b\x63\x6f\x6d\x6d\x61\x6e\x64\x4e\x61\x6d\x65\x5d\x0a\x0a\x54\x6f\x20\x73\x65\x65\x20\x6d\x6f\x72\x65\x20\x6f\x6e\x20\x65\x61\x63\x68\x20\x63\x6f\x6d\x6d\x61\x6e\x64\x20\x77\x69\x74\x68\x20\x73\x6f\x75\x72\x63\x65\x20\x61\x6e\x64\x20\x66\x75\x6c\x6c\x20\x64\x65\x73\x63\x72\x69\x70\x74\x69\x6f\x6e\x3a\x0a\x0a\x20\x20\x68\x65\x6c\x70\x20\x2d\x66\x20\x2d\x73\x20\x5b\x63\x6f\x6d\x6d\x61\x6e\x64\x4e\x61\x6d\x65\x5d\x0a\x0a\x7b\x7b\x20\x69\x66\x20\x6e\x6f\x74\x65\x71\x75\x61\x6c\x20\x28\x6c\x65\x6e\x20\x2e\x53\x75\x62\x73\x29\x20\x30\x7d\x7d\x54\x6f\x20\x73\x65\x65\x20\x6d\x6f\x72\x65\x20\x6f\x6e\x20\x65\x61\x63\x68\x20\x73\x75\x62\x63\x6f\x6d\x6d\x61\x6e\x64\x20\x63\x6f\x6d\x6d\x61\x6e\x64\x73\x3a\x0a\x0a\x20\x20\x68\x65\x6c\x70\x20\x5b\x73\x75\x62\x63\x6f\x6d\x6d\x61\x6e\x64\x5d\x20\x5b\x63\x6f\x6d\x6d\x61\x6e\x64\x4e\x61\x6d\x65\x5d\x0a\x0a\x20\x20\x68\x65\x6c\x70\x20\x2d\x66\x20\x2d\x73\x20\x5b\x73\x75\x62\x63\x6f\x6d\x6d\x61\x6e\x64\x5d\x20\x5b\x63\x6f\x6d\x6d\x61\x6e\x64\x4e\x61\x6d\x65\x5d\x7b\x7b\x65\x6e\x64\x7d\x7d\x0a")
	files["shogun-pkg-list.tml"] = []byte("\x53\x68\x6f\x67\x75\x6e\x20\x63\x6c\x61\x6e\x20\x7b\x7b\x20\x71\x75\x6f\x74\x65\x20\x2e\x4d\x61\x69\x6e\x2e\x50\x61\x63\x6b\x61\x67\x65\x7d\x7d\x0a\x0a\xe2\xa1\xbf\x20\x53\x41\x4d\x55\x52\x41\x49\x20\x7b\x7b\x20\x71\x75\x6f\x74\x65\x20\x2e\x4d\x61\x69\x6e\x2e\x4e\x61\x6d\x65\x20\x7d\x7d\x20\x20\x20\x20\x20\x20\x20\x20\x7b\x7b\x2e\x4d\x61\x69\x6e\x2e\x44\x65\x73\x63\x7d\x7d\x0a\x0a\x4b\x41\x54\x41\x4e\x41\x20\x43\x4f\x4d\x4d\x41\x4e\x44\x53\x3a\x0a\x7b\x7b\x20\x72\x61\x6e\x67\x65\x20\x24\x69\x6e\x64\x65\x78\x2c\x20\x24\x65\x6c\x65\x6d\x20\x3a\x3d\x20\x2e\x4d\x61\x69\x6e\x2e\x4c\x69\x73\x74\x7d\x7d\x7b\x7b\x20\x72\x61\x6e\x67\x65\x20\x24\x65\x6c\x65\x6d\x2e\x4c\x69\x73\x74\x7d\x7d\x0a\x20\x20\xe2\xa0\x99\x20\x7b\x7b\x20\x2e\x4e\x61\x6d\x65\x20\x7d\x7d\x7b\x7b\x24\x65\x6c\x65\x6d\x2e\x53\x70\x61\x63\x65\x46\x6f\x72\x20\x2e\x4e\x61\x6d\x65\x7d\x7d\x7b\x7b\x2e\x53\x79\x6e\x6f\x70\x73\x65\x73\x7d\x7d\x0a\x7b\x7b\x20\x65\x6e\x64\x20\x7d\x7d\x20\x7b\x7b\x20\x65\x6e\x64\x20\x7d\x7d\x0a\x0a\x4f\x54\x48\x45\x52\x20\x53\x41\x4d\x55\x52\x41\x49\x20\x43\x4f\x4d\x4d\x41\x4e\x44\x53\x3a\x0a\x0a\x7b\x7b\x20\x72\x61\x6e\x67\x65\x20\x24\x6e\x61\x6d\x65\x2c\x20\x24\x65\x6c\x65\x6d\x20\x3a\x3d\x20\x2e\x53\x75\x62\x73\x7d\x7d\x0a\xe2\xa1\xbf\x20\x7b\x7b\x20\x24\x65\x6c\x65\x6d\x2e\x4e\x61\x6d\x65\x20\x7d\x7d\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x7b\x7b\x24\x65\x6c\x65\x6d\x2e\x44\x65\x73\x63\x7d\x7d\x0a\x7b\x7b\x20\x65\x6e\x64\x20\x7d\x7d\x0a")
	files["shogun-src-pkg-content.tml"] = []byte("\x70\x61\x63\x6b\x61\x67\x65\x20\x70\x6b\x67\x0a\x0a\x7b\x7b\x2e\x53\x6f\x75\x72\x63\x65\x7d\x7d\x0a")
	files["shogun-src-pkg-hash.tml"] = []byte("\x7b\x7b\x2e\x48\x61\x73\x68\x7d\x7d\x0a")
	files["shogun-src-pkg-help-format.tml"] = []byte("\x4e\x41\x4d\x45\x3a\x0a\x7b\x7b\x2e\x4e\x61\x6d\x65\x7d\x7d\x20\x2d\x20\x7b\x7b\x2e\x55\x73\x61\x67\x65\x7d\x7d\x0a\x0a\x56\x45\x52\x53\x49\x4f\x4e\x3a\x0a\x7b\x7b\x2e\x56\x65\x72\x73\x69\x6f\x6e\x7d\x7d\x0a\x0a\x44\x45\x53\x43\x52\x49\x50\x54\x49\x4f\x4e\x3a\x0a\x7b\x7b\x2e\x44\x65\x73\x63\x72\x69\x70\x74\x69\x6f\x6e\x7d\x7d\x0a\x0a\x55\x53\x41\x47\x45\x3a\x0a\x7b\x7b\x2e\x4e\x61\x6d\x65\x7d\x7d\x20\x7b\x7b\x69\x66\x20\x2e\x46\x6c\x61\x67\x73\x7d\x7d\x5b\x66\x6c\x61\x67\x73\x5d\x20\x7b\x7b\x65\x6e\x64\x7d\x7d\x63\x6f\x6d\x6d\x61\x6e\x64\x7b\x7b\x69\x66\x20\x2e\x46\x6c\x61\x67\x73\x7d\x7d\x7b\x7b\x65\x6e\x64\x7d\x7d\x20\x5b\x61\x72\x67\x75\x6d\x65\x6e\x74\x73\x2e\x2e\x2e\x5d\x0a\x0a\x43\x4f\x4d\x4d\x41\x4e\x44\x53\x3a\x0a\x7b\x7b\x72\x61\x6e\x67\x65\x20\x2e\x43\x6f\x6d\x6d\x61\x6e\x64\x73\x7d\x7d\x7b\x7b\x6a\x6f\x69\x6e\x20\x2e\x4e\x61\x6d\x65\x73\x20\x22\x2c\x20\x22\x7d\x7d\x7b\x7b\x20\x22\x5c\x74\x22\x20\x7d\x7d\x7b\x7b\x2e\x55\x73\x61\x67\x65\x7d\x7d\x0a\x7b\x7b\x65\x6e\x64\x7d\x7d\x7b\x7b\x69\x66\x20\x2e\x46\x6c\x61\x67\x73\x7d\x7d\x0a\x46\x4c\x41\x47\x53\x3a\x0a\x7b\x7b\x72\x61\x6e\x67\x65\x20\x2e\x46\x6c\x61\x67\x73\x7d\x7d\x7b\x7b\x2e\x7d\x7d\x0a\x7b\x7b\x65\x6e\x64\x7d\x7d\x7b\x7b\x65\x6e\x64\x7d\x7d\x0a")
	files["shogun-src-pkg-main.tml"] = []byte("\x70\x61\x63\x6b\x61\x67\x65\x20\x6d\x61\x69\x6e\x0a\x0a\x69\x6d\x70\x6f\x72\x74\x20\x28\x0a\x09\x22\x66\x6d\x74\x22\x0a\x09\x22\x69\x6f\x22\x0a\x09\x22\x6f\x73\x22\x0a\x09\x22\x73\x74\x72\x69\x6e\x67\x73\x22\x0a\x0a\x09\x22\x67\x69\x74\x68\x75\x62\x2e\x63\x6f\x6d\x2f\x66\x61\x74\x69\x68\x2f\x63\x6f\x6c\x6f\x72\x22\x0a\x09\x22\x67\x69\x74\x68\x75\x62\x2e\x63\x6f\x6d\x2f\x6d\x69\x6e\x69\x6f\x2f\x63\x6c\x69\x22\x0a\x09\x7b\x7b\x71\x75\x6f\x74\x65\x20\x2e\x4d\x61\x69\x6e\x50\x61\x63\x6b\x61\x67\x65\x20\x7d\x7d\x0a\x29\x0a\x0a\x0a\x2f\x2f\x20\x76\x61\x72\x73\x20\x2e\x2e\x2e\x0a\x76\x61\x72\x20\x28\x0a\x20\x67\x72\x65\x65\x6e\x20\x3d\x20\x63\x6f\x6c\x6f\x72\x2e\x4e\x65\x77\x28\x63\x6f\x6c\x6f\x72\x2e\x46\x67\x47\x72\x65\x65\x6e\x29\x0a\x20\x62\x69\x6e\x48\x61\x73\x68\x20\x3d\x20\x7b\x7b\x71\x75\x6f\x74\x65\x20\x2e\x4d\x61\x69\x6e\x2e\x48\x61\x73\x68\x7d\x7d\x0a\x20\x62\x69\x6e\x4e\x61\x6d\x65\x20\x3d\x20\x7b\x7b\x71\x75\x6f\x74\x65\x20\x2e\x4d\x61\x69\x6e\x2e\x42\x69\x6e\x61\x72\x79\x4e\x61\x6d\x65\x20\x7d\x7d\x0a\x20\x56\x65\x72\x73\x69\x6f\x6e\x20\x3d\x20\x67\x72\x65\x65\x6e\x2e\x53\x70\x72\x69\x6e\x74\x66\x28\x22\x31\x2e\x30\x2e\x30\x22\x29\x0a\x20\x68\x65\x6c\x70\x4d\x65\x73\x73\x61\x67\x65\x20\x3d\x20\x73\x74\x72\x69\x6e\x67\x73\x2e\x54\x72\x69\x6d\x53\x70\x61\x63\x65\x28\x60\x7b\x7b\x2e\x48\x65\x6c\x70\x46\x6f\x72\x6d\x61\x74\x20\x7d\x7d\x60\x29\x0a\x20\x63\x75\x73\x74\x6f\x6d\x48\x65\x6c\x70\x54\x65\x6d\x70\x6c\x61\x74\x65\x20\x3d\x20\x60\x7b\x7b\x2e\x43\x75\x73\x74\x6f\x6d\x48\x65\x6c\x70\x54\x65\x6d\x70\x6c\x61\x74\x65\x7d\x7d\x60\x0a\x29\x0a\x0a\x66\x75\x6e\x63\x20\x6d\x61\x69\x6e\x28\x29\x7b\x0a\x09\x61\x70\x70\x20\x3a\x3d\x20\x63\x6c\x69\x2e\x4e\x65\x77\x41\x70\x70\x28\x29\x0a\x09\x61\x70\x70\x2e\x4e\x61\x6d\x65\x20\x3d\x20\x22\x7b\x7b\x6c\x6f\x77\x65\x72\x20\x2e\x4d\x61\x69\x6e\x2e\x42\x69\x6e\x61\x72\x79\x4e\x61\x6d\x65\x7d\x7d\x22\x0a\x09\x61\x70\x70\x2e\x55\x73\x61\x67\x65\x20\x3d\x20\x22\x7b\x7b\x6c\x6f\x77\x65\x72\x20\x2e\x4d\x61\x69\x6e\x2e\x42\x69\x6e\x61\x72\x79\x4e\x61\x6d\x65\x7d\x7d\x20\x5b\x63\x6f\x6d\x6d\x61\x6e\x64\x5d\x22\x0a\x20\x20\x61\x70\x70\x2e\x56\x65\x72\x73\x69\x6f\x6e\x20\x3d\x20\x56\x65\x72\x73\x69\x6f\x6e\x0a\x09\x61\x70\x70\x2e\x43\x75\x73\x74\x6f\x6d\x41\x70\x70\x48\x65\x6c\x70\x54\x65\x6d\x70\x6c\x61\x74\x65\x20\x3d\x20\x68\x65\x6c\x70\x4d\x65\x73\x73\x61\x67\x65\x0a\x09\x61\x70\x70\x2e\x44\x65\x73\x63\x72\x69\x70\x74\x69\x6f\x6e\x20\x3d\x20\x22\x7b\x7b\x6c\x6f\x77\x65\x72\x20\x2e\x4d\x61\x69\x6e\x2e\x42\x69\x6e\x61\x72\x79\x4e\x61\x6d\x65\x7d\x7d\x20\x67\x65\x6e\x65\x72\x61\x74\x65\x64\x20\x62\x79\x20\x73\x68\x6f\x67\x75\x6e\x22\x0a\x20\x20\x61\x70\x70\x2e\x41\x63\x74\x69\x6f\x6e\x20\x3d\x20\x6d\x61\x69\x6e\x41\x63\x74\x69\x6f\x6e\x0a\x0a\x0a\x09\x61\x70\x70\x2e\x43\x6f\x6d\x6d\x61\x6e\x64\x73\x20\x3d\x20\x5b\x5d\x63\x6c\x69\x2e\x43\x6f\x6d\x6d\x61\x6e\x64\x7b\x0a\x09\x09\x7b\x0a\x09\x09\x09\x4e\x61\x6d\x65\x3a\x20\x20\x20\x22\x68\x65\x6c\x70\x22\x2c\x0a\x09\x09\x09\x41\x63\x74\x69\x6f\x6e\x3a\x20\x68\x65\x6c\x70\x41\x63\x74\x69\x6f\x6e\x2c\x0a\x09\x09\x09\x46\x6c\x61\x67\x73\x3a\x20\x20\x5b\x5d\x63\x6c\x69\x2e\x46\x6c\x61\x67\x7b\x0a\x09\x09\x09\x09\x63\x6c\x69\x2e\x42\x6f\x6f\x6c\x46\x6c\x61\x67\x7b\x0a\x09\x09\x09\x09\x09\x4e\x61\x6d\x65\x3a\x20\x20\x22\x73\x2c\x73\x6f\x75\x72\x63\x65\x22\x2c\x0a\x09\x09\x09\x09\x09\x55\x73\x61\x67\x65\x3a\x20\x22\x2d\x73\x6f\x75\x72\x63\x65\x20\x74\x6f\x20\x73\x68\x6f\x77\x20\x73\x6f\x75\x72\x63\x65\x20\x6f\x66\x20\x63\x6f\x6d\x6d\x61\x6e\x64\x20\x61\x73\x20\x77\x65\x6c\x6c\x22\x2c\x0a\x09\x09\x09\x09\x7d\x2c\x0a\x09\x09\x09\x09\x63\x6c\x69\x2e\x42\x6f\x6f\x6c\x46\x6c\x61\x67\x7b\x0a\x09\x09\x09\x09\x09\x4e\x61\x6d\x65\x3a\x20\x20\x22\x66\x2c\x66\x75\x6c\x6c\x22\x2c\x0a\x09\x09\x09\x09\x09\x55\x73\x61\x67\x65\x3a\x20\x22\x2d\x66\x75\x6c\x6c\x20\x74\x6f\x20\x73\x68\x6f\x77\x20\x66\x75\x6c\x6c\x20\x64\x65\x73\x63\x72\x69\x70\x74\x69\x6f\x6e\x20\x6f\x66\x20\x63\x6f\x6d\x6d\x61\x6e\x64\x22\x2c\x0a\x09\x09\x09\x09\x7d\x2c\x0a\x09\x09\x09\x7d\x2c\x0a\x09\x09\x7d\x2c\x0a\x09\x7d\x0a\x0a\x09\x61\x70\x70\x2e\x52\x75\x6e\x41\x6e\x64\x45\x78\x69\x74\x4f\x6e\x45\x72\x72\x6f\x72\x28\x29\x0a\x7d\x0a\x0a\x66\x75\x6e\x63\x20\x68\x65\x6c\x70\x41\x63\x74\x69\x6f\x6e\x28\x63\x20\x2a\x63\x6c\x69\x2e\x43\x6f\x6e\x74\x65\x78\x74\x29\x20\x65\x72\x72\x6f\x72\x20\x7b\x0a\x09\x69\x66\x20\x63\x2e\x4e\x41\x72\x67\x28\x29\x20\x3d\x3d\x20\x30\x20\x7b\x0a\x09\x09\x20\x66\x6d\x74\x2e\x50\x72\x69\x6e\x74\x6c\x6e\x28\x68\x65\x6c\x70\x4d\x65\x73\x73\x61\x67\x65\x29\x0a\x09\x20\x72\x65\x74\x75\x72\x6e\x20\x6e\x69\x6c\x0a\x09\x7d\x0a\x0a\x09\x72\x65\x74\x75\x72\x6e\x20\x70\x6b\x67\x2e\x4d\x61\x69\x6e\x53\x68\x6f\x67\x75\x6e\x48\x65\x6c\x70\x28\x0a\x09\x09\x63\x2e\x42\x6f\x6f\x6c\x28\x22\x73\x6f\x75\x72\x63\x65\x22\x29\x2c\x0a\x09\x09\x63\x2e\x42\x6f\x6f\x6c\x28\x22\x66\x75\x6c\x6c\x22\x29\x2c\x0a\x09\x09\x63\x2e\x41\x72\x67\x73\x28\x29\x2e\x46\x69\x72\x73\x74\x28\x29\x2c\x0a\x09\x09\x63\x2e\x41\x72\x67\x73\x28\x29\x2e\x54\x61\x69\x6c\x28\x29\x2c\x0a\x09\x09\x6f\x73\x2e\x53\x74\x64\x69\x6e\x2c\x0a\x09\x09\x77\x6f\x70\x43\x6c\x6f\x73\x65\x72\x7b\x57\x72\x69\x74\x65\x72\x3a\x20\x6f\x73\x2e\x53\x74\x64\x6f\x75\x74\x7d\x2c\x0a\x09\x29\x0a\x7d\x0a\x0a\x66\x75\x6e\x63\x20\x6d\x61\x69\x6e\x41\x63\x74\x69\x6f\x6e\x28\x63\x20\x2a\x63\x6c\x69\x2e\x43\x6f\x6e\x74\x65\x78\x74\x29\x20\x65\x72\x72\x6f\x72\x20\x7b\x0a\x09\x69\x66\x20\x63\x2e\x4e\x41\x72\x67\x28\x29\x20\x3d\x3d\x20\x30\x20\x7b\x0a\x09\x09\x20\x66\x6d\x74\x2e\x50\x72\x69\x6e\x74\x6c\x6e\x28\x68\x65\x6c\x70\x4d\x65\x73\x73\x61\x67\x65\x29\x0a\x09\x20\x72\x65\x74\x75\x72\x6e\x20\x6e\x69\x6c\x0a\x09\x7d\x0a\x0a\x09\x72\x65\x74\x75\x72\x6e\x20\x70\x6b\x67\x2e\x4d\x61\x69\x6e\x53\x68\x6f\x67\x75\x6e\x45\x78\x65\x63\x75\x74\x65\x28\x0a\x09\x09\x63\x2e\x41\x72\x67\x73\x28\x29\x2e\x46\x69\x72\x73\x74\x28\x29\x2c\x0a\x09\x09\x63\x2e\x41\x72\x67\x73\x28\x29\x2e\x54\x61\x69\x6c\x28\x29\x2c\x0a\x09\x09\x6f\x73\x2e\x53\x74\x64\x69\x6e\x2c\x0a\x09\x09\x77\x6f\x70\x43\x6c\x6f\x73\x65\x72\x7b\x57\x72\x69\x74\x65\x72\x3a\x20\x6f\x73\x2e\x53\x74\x64\x6f\x75\x74\x7d\x2c\x0a\x09\x29\x0a\x7d\x0a\x0a\x74\x79\x70\x65\x20\x77\x6f\x70\x43\x6c\x6f\x73\x65\x72\x20\x73\x74\x72\x75\x63\x74\x7b\x0a\x09\x69\x6f\x2e\x57\x72\x69\x74\x65\x72\x0a\x7d\x0a\x0a\x2f\x2f\x20\x43\x6c\x6f\x73\x65\x20\x64\x6f\x65\x73\x20\x6e\x6f\x74\x68\x69\x6e\x67\x2e\x0a\x66\x75\x6e\x63\x20\x28\x77\x6f\x70\x43\x6c\x6f\x73\x65\x72\x29\x20\x43\x6c\x6f\x73\x65\x28\x29\x20\x65\x72\x72\x6f\x72\x20\x7b\x0a\x09\x72\x65\x74\x75\x72\x6e\x20\x6e\x69\x6c\x0a\x7d\x0a")
	files["shogun-src-pkg.tml"] = []byte("\x2f\x2f\x20\x57\x41\x52\x4e\x49\x4e\x47\x3a\x20\x44\x6f\x20\x6e\x6f\x74\x20\x65\x64\x69\x74\x2c\x20\x74\x68\x69\x73\x20\x66\x69\x6c\x65\x20\x69\x73\x20\x61\x75\x74\x6f\x67\x65\x6e\x65\x72\x61\x74\x65\x64\x2e\x0a\x0a\x70\x61\x63\x6b\x61\x67\x65\x20\x70\x6b\x67\x0a\x0a\x69\x6d\x70\x6f\x72\x74\x20\x28\x0a\x20\x20\x22\x69\x6f\x22\x0a\x7b\x7b\x20\x72\x61\x6e\x67\x65\x20\x24\x5f\x2c\x20\x24\x73\x75\x62\x20\x3a\x3d\x20\x2e\x53\x75\x62\x73\x7d\x7d\x0a\x20\x20\x7b\x7b\x24\x73\x75\x62\x2e\x43\x6c\x65\x61\x6e\x42\x69\x6e\x61\x72\x79\x4e\x61\x6d\x65\x7d\x7d\x20\x7b\x7b\x71\x75\x6f\x74\x65\x20\x24\x73\x75\x62\x2e\x50\x6b\x67\x50\x61\x74\x68\x7d\x7d\x0a\x7b\x7b\x65\x6e\x64\x7d\x7d\x0a\x29\x0a\x0a\x76\x61\x72\x20\x28\x0a\x20\x20\x73\x75\x62\x43\x6f\x6d\x6d\x61\x6e\x64\x73\x20\x3d\x20\x6d\x61\x70\x5b\x73\x74\x72\x69\x6e\x67\x5d\x62\x6f\x6f\x6c\x7b\x20\x7b\x7b\x20\x72\x61\x6e\x67\x65\x20\x24\x5f\x2c\x20\x24\x73\x75\x62\x20\x3a\x3d\x20\x2e\x53\x75\x62\x73\x7d\x7d\x0a\x20\x20\x20\x20\x7b\x7b\x20\x71\x75\x6f\x74\x65\x20\x24\x73\x75\x62\x2e\x42\x69\x6e\x61\x72\x79\x4e\x61\x6d\x65\x7d\x7d\x3a\x20\x74\x72\x75\x65\x2c\x0a\x7b\x7b\x65\x6e\x64\x7d\x7d\x20\x7d\x0a\x29\x0a\x0a\x2f\x2f\x20\x4d\x61\x69\x6e\x53\x68\x6f\x67\x75\x6e\x45\x78\x65\x63\x75\x74\x65\x20\x65\x78\x65\x63\x75\x74\x65\x73\x20\x6e\x65\x63\x65\x73\x73\x61\x72\x79\x20\x63\x6f\x6d\x6d\x61\x6e\x64\x73\x20\x61\x73\x20\x6e\x65\x65\x64\x65\x64\x20\x66\x72\x6f\x6d\x20\x69\x74\x73\x20\x61\x72\x67\x75\x6d\x65\x6e\x74\x73\x20\x61\x6e\x64\x0a\x2f\x2f\x20\x77\x72\x69\x74\x65\x73\x20\x63\x6f\x72\x72\x65\x73\x70\x6f\x6e\x64\x69\x6e\x67\x20\x6f\x75\x74\x70\x75\x74\x73\x20\x74\x6f\x20\x70\x72\x6f\x76\x69\x64\x65\x64\x20\x60\x6f\x75\x67\x6f\x69\x6e\x67\x60\x20\x77\x72\x69\x74\x65\x43\x6c\x6f\x73\x65\x72\x2e\x0a\x66\x75\x6e\x63\x20\x4d\x61\x69\x6e\x53\x68\x6f\x67\x75\x6e\x45\x78\x65\x63\x75\x74\x65\x28\x63\x6d\x64\x20\x73\x74\x72\x69\x6e\x67\x2c\x20\x61\x72\x67\x73\x20\x5b\x5d\x73\x74\x72\x69\x6e\x67\x2c\x20\x69\x6e\x63\x6f\x6d\x69\x6e\x67\x20\x69\x6f\x2e\x52\x65\x61\x64\x65\x72\x2c\x20\x6f\x75\x74\x67\x6f\x69\x6e\x67\x20\x69\x6f\x2e\x57\x72\x69\x74\x65\x43\x6c\x6f\x73\x65\x72\x29\x20\x65\x72\x72\x6f\x72\x20\x7b\x0a\x20\x20\x7b\x7b\x20\x69\x66\x20\x6e\x6f\x74\x65\x71\x75\x61\x6c\x20\x28\x6c\x65\x6e\x20\x2e\x53\x75\x62\x73\x29\x20\x30\x7d\x7d\x2f\x2f\x20\x49\x66\x20\x69\x74\x73\x20\x61\x20\x73\x75\x62\x63\x6f\x6d\x6d\x61\x6e\x64\x20\x74\x68\x65\x6e\x20\x6c\x65\x74\x20\x73\x75\x62\x63\x6f\x6d\x6d\x61\x6e\x64\x20\x68\x61\x6e\x64\x6c\x65\x20\x74\x68\x69\x73\x2e\x0a\x20\x20\x69\x66\x20\x73\x75\x62\x43\x6f\x6d\x6d\x61\x6e\x64\x73\x5b\x63\x6d\x64\x5d\x20\x7b\x0a\x20\x20\x20\x20\x76\x61\x72\x20\x66\x69\x72\x73\x74\x20\x73\x74\x72\x69\x6e\x67\x0a\x20\x20\x20\x20\x76\x61\x72\x20\x72\x65\x73\x74\x20\x5b\x5d\x73\x74\x72\x69\x6e\x67\x0a\x0a\x20\x20\x20\x20\x69\x66\x20\x6c\x65\x6e\x28\x61\x72\x67\x73\x29\x20\x21\x3d\x20\x30\x20\x7b\x0a\x20\x20\x20\x20\x20\x20\x66\x69\x72\x73\x74\x20\x3d\x20\x61\x72\x67\x73\x5b\x30\x5d\x0a\x20\x20\x20\x20\x20\x20\x72\x65\x73\x74\x20\x3d\x20\x61\x72\x67\x73\x5b\x31\x3a\x5d\x0a\x20\x20\x20\x20\x7d\x0a\x0a\x20\x20\x20\x20\x73\x77\x69\x74\x63\x68\x20\x63\x6d\x64\x20\x7b\x0a\x20\x20\x20\x20\x7b\x7b\x20\x72\x61\x6e\x67\x65\x20\x24\x5f\x2c\x20\x24\x73\x75\x62\x20\x3a\x3d\x20\x2e\x53\x75\x62\x73\x7d\x7d\x0a\x20\x20\x20\x20\x20\x20\x63\x61\x73\x65\x20\x7b\x7b\x71\x75\x6f\x74\x65\x20\x24\x73\x75\x62\x2e\x42\x69\x6e\x61\x72\x79\x4e\x61\x6d\x65\x7d\x7d\x3a\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x72\x65\x74\x75\x72\x6e\x20\x7b\x7b\x24\x73\x75\x62\x2e\x43\x6c\x65\x61\x6e\x42\x69\x6e\x61\x72\x79\x4e\x61\x6d\x65\x7d\x7d\x2e\x4d\x61\x69\x6e\x53\x68\x6f\x67\x75\x6e\x45\x78\x65\x63\x75\x74\x65\x28\x66\x69\x72\x73\x74\x2c\x20\x72\x65\x73\x74\x2c\x20\x69\x6e\x63\x6f\x6d\x69\x6e\x67\x2c\x20\x6f\x75\x74\x67\x6f\x69\x6e\x67\x29\x0a\x20\x20\x20\x20\x7b\x7b\x65\x6e\x64\x7d\x7d\x0a\x20\x20\x20\x20\x7d\x0a\x20\x20\x7d\x0a\x20\x20\x7b\x7b\x65\x6e\x64\x7d\x7d\x73\x77\x69\x74\x63\x68\x20\x63\x6d\x64\x20\x7b\x0a\x20\x20\x20\x20\x7b\x7b\x20\x72\x61\x6e\x67\x65\x20\x24\x5f\x2c\x20\x24\x65\x6c\x65\x6d\x20\x3a\x3d\x20\x2e\x4d\x61\x69\x6e\x2e\x46\x75\x6e\x63\x74\x69\x6f\x6e\x73\x20\x7d\x7d\x7b\x7b\x72\x61\x6e\x67\x65\x20\x24\x65\x6c\x65\x6d\x2e\x4c\x69\x73\x74\x7d\x7d\x0a\x20\x20\x20\x20\x20\x20\x63\x61\x73\x65\x20\x7b\x7b\x71\x75\x6f\x74\x65\x20\x2e\x4e\x61\x6d\x65\x7d\x7d\x2c\x20\x7b\x7b\x71\x75\x6f\x74\x65\x20\x2e\x52\x65\x61\x6c\x4e\x61\x6d\x65\x7d\x7d\x3a\x0a\x20\x20\x20\x20\x7b\x7b\x65\x6e\x64\x7d\x7d\x7b\x7b\x20\x65\x6e\x64\x20\x7d\x7d\x0a\x20\x20\x7d\x0a\x20\x20\x72\x65\x74\x75\x72\x6e\x20\x6e\x69\x6c\x0a\x7d\x0a\x0a\x2f\x2f\x20\x4d\x61\x69\x6e\x53\x68\x6f\x67\x75\x6e\x48\x65\x6c\x70\x20\x64\x69\x73\x70\x6c\x61\x79\x20\x68\x65\x6c\x70\x20\x6d\x65\x73\x73\x61\x67\x65\x20\x66\x6f\x72\x20\x65\x78\x65\x63\x75\x74\x61\x62\x6c\x65\x20\x63\x6f\x6d\x6d\x61\x6e\x64\x73\x20\x61\x6e\x64\x20\x73\x75\x62\x63\x6f\x6d\x6d\x61\x6e\x64\x73\x2e\x0a\x66\x75\x6e\x63\x20\x4d\x61\x69\x6e\x53\x68\x6f\x67\x75\x6e\x48\x65\x6c\x70\x28\x73\x6f\x75\x72\x63\x65\x20\x62\x6f\x6f\x6c\x2c\x20\x66\x75\x6c\x6c\x64\x65\x73\x63\x20\x62\x6f\x6f\x6c\x2c\x20\x63\x6d\x64\x20\x73\x74\x72\x69\x6e\x67\x2c\x20\x61\x72\x67\x73\x20\x5b\x5d\x73\x74\x72\x69\x6e\x67\x2c\x20\x69\x6e\x63\x6f\x6d\x69\x6e\x67\x20\x69\x6f\x2e\x52\x65\x61\x64\x65\x72\x2c\x20\x6f\x75\x74\x67\x6f\x69\x6e\x67\x20\x69\x6f\x2e\x57\x72\x69\x74\x65\x43\x6c\x6f\x73\x65\x72\x29\x20\x65\x72\x72\x6f\x72\x20\x7b\x0a\x20\x20\x7b\x7b\x20\x69\x66\x20\x6e\x6f\x74\x65\x71\x75\x61\x6c\x20\x28\x6c\x65\x6e\x20\x2e\x53\x75\x62\x73\x29\x20\x30\x7d\x7d\x2f\x2f\x20\x49\x66\x20\x69\x74\x73\x20\x61\x20\x73\x75\x62\x63\x6f\x6d\x6d\x61\x6e\x64\x20\x74\x68\x65\x6e\x20\x6c\x65\x74\x20\x73\x75\x62\x63\x6f\x6d\x6d\x61\x6e\x64\x20\x68\x61\x6e\x64\x6c\x65\x20\x74\x68\x69\x73\x2e\x0a\x20\x20\x69\x66\x20\x73\x75\x62\x43\x6f\x6d\x6d\x61\x6e\x64\x73\x5b\x63\x6d\x64\x5d\x20\x7b\x0a\x20\x20\x20\x20\x76\x61\x72\x20\x66\x69\x72\x73\x74\x20\x73\x74\x72\x69\x6e\x67\x0a\x20\x20\x20\x20\x76\x61\x72\x20\x72\x65\x73\x74\x20\x5b\x5d\x73\x74\x72\x69\x6e\x67\x0a\x0a\x20\x20\x20\x20\x69\x66\x20\x6c\x65\x6e\x28\x61\x72\x67\x73\x29\x20\x21\x3d\x20\x30\x20\x7b\x0a\x20\x20\x20\x20\x20\x20\x66\x69\x72\x73\x74\x20\x3d\x20\x61\x72\x67\x73\x5b\x30\x5d\x0a\x20\x20\x20\x20\x20\x20\x72\x65\x73\x74\x20\x3d\x20\x61\x72\x67\x73\x5b\x31\x3a\x5d\x0a\x20\x20\x20\x20\x7d\x0a\x0a\x20\x20\x20\x20\x73\x77\x69\x74\x63\x68\x20\x63\x6d\x64\x20\x7b\x0a\x20\x20\x20\x20\x7b\x7b\x20\x72\x61\x6e\x67\x65\x20\x24\x5f\x2c\x20\x24\x73\x75\x62\x20\x3a\x3d\x20\x2e\x53\x75\x62\x73\x7d\x7d\x0a\x20\x20\x20\x20\x20\x20\x63\x61\x73\x65\x20\x7b\x7b\x71\x75\x6f\x74\x65\x20\x24\x73\x75\x62\x2e\x42\x69\x6e\x61\x72\x79\x4e\x61\x6d\x65\x7d\x7d\x3a\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x72\x65\x74\x75\x72\x6e\x20\x7b\x7b\x24\x73\x75\x62\x2e\x43\x6c\x65\x61\x6e\x42\x69\x6e\x61\x72\x79\x4e\x61\x6d\x65\x7d\x7d\x2e\x4d\x61\x69\x6e\x53\x68\x6f\x67\x75\x6e\x48\x65\x6c\x70\x28\x73\x6f\x75\x72\x63\x65\x2c\x20\x66\x75\x6c\x6c\x64\x65\x73\x63\x2c\x66\x69\x72\x73\x74\x2c\x20\x72\x65\x73\x74\x2c\x20\x69\x6e\x63\x6f\x6d\x69\x6e\x67\x2c\x20\x6f\x75\x74\x67\x6f\x69\x6e\x67\x29\x0a\x20\x20\x20\x20\x7b\x7b\x65\x6e\x64\x7d\x7d\x0a\x20\x20\x20\x20\x7d\x0a\x20\x20\x7d\x0a\x20\x20\x7b\x7b\x65\x6e\x64\x7d\x7d\x73\x77\x69\x74\x63\x68\x20\x63\x6d\x64\x20\x7b\x0a\x20\x20\x20\x20\x7b\x7b\x20\x72\x61\x6e\x67\x65\x20\x24\x5f\x2c\x20\x24\x65\x6c\x65\x6d\x20\x3a\x3d\x20\x2e\x4d\x61\x69\x6e\x2e\x46\x75\x6e\x63\x74\x69\x6f\x6e\x73\x20\x7d\x7d\x7b\x7b\x72\x61\x6e\x67\x65\x20\x24\x65\x6c\x65\x6d\x2e\x4c\x69\x73\x74\x7d\x7d\x0a\x20\x20\x20\x20\x20\x20\x63\x61\x73\x65\x20\x7b\x7b\x71\x75\x6f\x74\x65\x20\x2e\x4e\x61\x6d\x65\x7d\x7d\x2c\x20\x7b\x7b\x71\x75\x6f\x74\x65\x20\x2e\x52\x65\x61\x6c\x4e\x61\x6d\x65\x7d\x7d\x3a\x0a\x20\x20\x20\x20\x7b\x7b\x65\x6e\x64\x7d\x7d\x7b\x7b\x20\x65\x6e\x64\x20\x7d\x7d\x0a\x20\x20\x7d\x0a\x20\x20\x72\x65\x74\x75\x72\x6e\x20\x6e\x69\x6c\x0a\x7d\x0a")

}
