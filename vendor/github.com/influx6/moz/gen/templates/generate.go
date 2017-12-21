// +build ignore

package main

import (
	"fmt"
	"os"
	"bytes"
	"strings"
	"path/filepath"

	"github.com/gu-io/gu/parsers/otherfiles"
)

var pkgName = "templates"
var pkg = "// Package {{PKGNAME}} is an auto-generated package which exposes the specific  \n// functionalities needed as desired to the specific reason which this package \n// exists for. Feel free to change this description.\n\n//go:generate go run generate.go\n\npackage {{PKG}}\n\nimport (\n\t\"fmt\"\n)\n\nvar internalFiles = map[string]string{}\n\n\n// Must retrieves the giving file and the content of that giving file else \n// panics if not found.\nfunc Must(file string) string {\n\tif content, ok := Get(file); ok {\n\t\treturn content\n\t}\n\t\n\tpanic(fmt.Sprintf(\"File %s not found\", file))\n}\n\n\n// Get retrieves the giving file and the content of that giving file.\nfunc Get(file string) (string, bool) {\n\titem, ok := internalFiles[file]\n\treturn item, ok\n}\n\nfunc init(){\n{{FILES}}\n}"

func main() {
	items, err := otherfiles.ParseDir(filepath.Join("./", "ast"), []string{".tml"})
	if err != nil {
		panic("Failed to walk html files: "+ err.Error())
	}

    var buf bytes.Buffer

    for path, item := range items {
        fmt.Fprintf(&buf,"\tinternalFiles[%q] = %+q\n", path, item)
    }

	file, err := os.Create(filepath.Join("./", "templates.go"))
	if err != nil {
		panic("Failed to create css pkg file: "+ err.Error())
	}

	defer file.Close()

	pkg = strings.Replace(pkg,"{{PKG}}", pkgName, -1)
	pkg = strings.Replace(pkg,"{{FILES}}", buf.String(), -1)

	file.Write([]byte(pkg))
}