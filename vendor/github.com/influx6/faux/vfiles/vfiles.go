package vfiles

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"

	"runtime"

	"github.com/influx6/faux/hexwriter"
)

// ParseDir returns a new instance of all files located within the provided directory.
func ParseDir(dir string, allowedExtensions []string) (map[string]string, error) {
	items := make(map[string]string)

	// Walk directory pulling contents into  items.
	if cerr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if cerr := walkDir(allowedExtensions, items, dir, path, info, err); cerr != nil {
			return cerr
		}

		return nil
	}); cerr != nil {
		return nil, cerr
	}

	return items, nil
}

// validExension returns true/false if the extension provide is a valid acceptable one
// based on the allowedExtensions string slice.
func validExtension(extensions []string, ext string) bool {
	for _, es := range extensions {
		if es != ext {
			continue
		}

		return true
	}

	return false
}

// walkDir adds the giving path if it matches certain criterias into the items map.
func walkDir(extensions []string, items map[string]string, root string, path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if !info.Mode().IsRegular() {
		return nil
	}

	// Is file an exension we allow else skip.
	if len(extensions) != 0 && !validExtension(extensions, filepath.Ext(path)) {
		return nil
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}

	relFile, err := os.Open(path)
	if err != nil {
		return err
	}

	defer relFile.Close()

	var contents bytes.Buffer

	io.Copy(hexwriter.New(&contents), relFile)

	if runtime.GOOS == "windows" {
		items[filepath.ToSlash(rel)] = contents.String()
	} else {
		items[rel] = contents.String()
	}

	return nil
}

//========================================================================================

var errStopWalking = errors.New("stop walking directory")

// DirWalker defines a function type which for processing a path and it's info
// retrieved from the fs.
type DirWalker func(rel string, abs string, info os.FileInfo) error

// WalkDir will run through the provided path which is expected to be a directory
// and runs the provided callback with the current path and FileInfo.
func WalkDir(dir string, callback DirWalker) error {
	isWin := runtime.GOOS == "windows"

	cerr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		// If we got an error then stop and return it.
		if err != nil {
			return err
		}

		// If its a symlink, don't deal with it.
		if !info.Mode().IsRegular() {
			return nil
		}

		// If on windows, correct path slash.
		if isWin {
			path = filepath.ToSlash(path)
		}

		// Retrive relative path for giving path.
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		// If false is return then stop walking and return errStopWalking.
		if err := callback(relPath, path, info); err != nil {
			return err
		}

		return nil
	})

	// If we received error to stop walking then skip
	if cerr == errStopWalking {
		return nil
	}

	return cerr
}

//========================================================================================

var isWin = (runtime.GOOS == "windows")

// WalkDirSurface walks the directory and it's children but does not look beyond the first level.
// It only runs the callback against the content but will not run deeper into subdirectories of root.
func WalkDirSurface(dirpath string, callback DirWalker) error {
	dir, err := os.Open(dirpath)
	if err != nil {
		return err
	}

	dirInfo, err := dir.Stat()
	if err != nil {
		return err
	}

	if !dirInfo.IsDir() {
		return errors.New("Only directories allowed")
	}

	fileInfos, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	for _, info := range fileInfos {
		infoPath := filepath.Join(dirpath, info.Name())

		// If on windows, correct path slash.
		if isWin {
			infoPath = filepath.ToSlash(infoPath)
		}

		// If false is return then stop walking and return errStopWalking.
		if err := callback(info.Name(), infoPath, info); err != nil {
			return err
		}
	}

	return nil
}
