package srcpath

import (
	"os"
	"path/filepath"
)

var (
	goPath    = os.Getenv("GOPATH")
	goSrcPath = filepath.Join(goPath, "src")
)

// SrcPath returns current go src path.
func SrcPath() string {
	return goSrcPath
}

// FromSrcPath returns the giving path as absolute from the gosrc path.
func FromSrcPath(pr string) string {
	return filepath.Join(goSrcPath, pr)
}

// RelativeToSrc returns a path that is relative to the go src path.
func RelativeToSrc(path string) (string, error) {
	return filepath.Rel(goSrcPath, path)
}
