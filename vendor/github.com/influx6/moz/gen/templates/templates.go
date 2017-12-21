// Package {{PKGNAME}} is an auto-generated package which exposes the specific  
// functionalities needed as desired to the specific reason which this package 
// exists for. Feel free to change this description.

//go:generate go run generate.go

package templates

import (
	"fmt"
)

var internalFiles = map[string]string{}


// Must retrieves the giving file and the content of that giving file else 
// panics if not found.
func Must(file string) string {
	if content, ok := Get(file); ok {
		return content
	}
	
	panic(fmt.Sprintf("File %s not found", file))
}


// Get retrieves the giving file and the content of that giving file.
func Get(file string) (string, bool) {
	item, ok := internalFiles[file]
	return item, ok
}

func init(){
	internalFiles["name.tml"] = "{{.Name}}"
	internalFiles["import.tml"] = "import ({{ range .Imports}}\n    {{.}}\n{{end}})\n\n"
	internalFiles["json.tml"] = "{\n{{ $len := subtract (len .Documents) 1 }}\n{{ range $ind, $item := .Documents }}\n    {{ if lessThan $ind $len}}{{$item}},{{else}}{{$item}}{{end}}\n{{ end }}\n}"
	internalFiles["typename.tml"] = "{{.Type}}"
	internalFiles["var-variable-type.tml"] = "var {{.Name}} {{.Type}}\n"
	internalFiles["variable-assign-basic.tml"] = "var {{.Name}} = {{.Value}}\n"
	internalFiles["variable-type-only.tml"] = "{{.Type}}"
	internalFiles["case.tml"] = "case {{.Condition}}:\n    {{.Action}}\n\n\n "
	internalFiles["comments.tml"] = "// {{.MainBlock}}\n// {{ range .Blocks}}\n// {{.}}\n// {{end}}\n//\n"
	internalFiles["function-type.tml"] = "func {{.Name}}{{.Constructor}} {{.Returns}}"
	internalFiles["jsonblock.tml"] = "{\n{{ $len := subtract (len .) 1 }}\n{{ range $ind, $item := . }}\n    {{ if lessThan $ind $len}}{{$item}},{{else}}{{$item}}{{end}}\n{{ end }}\n}"
	internalFiles["struct.tml"] = "{{.Comments}}\n{{.Annotations}}\ntype {{.Name}} {{.Type}} {\n{{ range .Fields }}\n    {{.}} \n{{ end }}\n}"
	internalFiles["value.tml"] = "{{.Value}}"
	internalFiles["variable-assign.tml"] = "{{.Name}}:={{.Value}}\n"
	internalFiles["map.tml"] = "{{.MapType}}[{{.Type}}]{{.Value}}{\n    {{ range $k, $v :=  .Values }}\n        {{quote $k}}: {{$v}},\n    {{ end }}\n}"
	internalFiles["package.tml"] = "{{ if notequal .Name \"\" }}package {{.Name}}\n{{ end }}\n{{.Body}}"
	internalFiles["structtype.tml"] = "{{.Name}} {{.Type}} {{.Tags}}"
	internalFiles["variable-type.tml"] = "{{.Name}} {{.Type}}\n"
	internalFiles["array.tml"] = "[{{.Size}}]{{.Type}}"
	internalFiles["function.tml"] = "\nfunc {{.Name}}{{.Constructor}} {{.Returns}} {\n{{.Body}}\n}\n"
	internalFiles["if.tml"] = "if{{.Condition}}{\n{{.Action}}\n}"
	internalFiles["annotations.tml"] = "//@{{.Value}}"
	internalFiles["case-default.tml"] = "default:\n    {{.Action}}\n\n\n "
	internalFiles["import-item.tml"] = "{{.Namespace}} \"{{.Path}}\"\n"
	internalFiles["map-header.tml"] = "{{.MapType}}[{{.Type}}]{{.ValueType}}"
	internalFiles["slicevalue.tml"] = "[]{{.Type}}{ {{.Values}} }"
	internalFiles["tag.tml"] = "{{.Format}}:\"{{.Name}}\""
	internalFiles["value-assign.tml"] = "{{.Name}} = {{.Value}}\n"
	internalFiles["variable-name.tml"] = "{{.Name}}"
	internalFiles["multicomments.tml"] = "/* {{.MainBlock}}\n{{ range .Blocks}}\n* {{.}}\n{{end}}\n*/\n"
	internalFiles["slicetype.tml"] = "[]{{.Type}}"
	internalFiles["switch.tml"] = "switch {{.Condition}} {\n{{.Case }}\n{{.Default }}\n}"
	internalFiles["text.tml"] = "{{.Block}}"

}