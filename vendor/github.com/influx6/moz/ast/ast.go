package ast

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/icrowley/fake"
	"github.com/influx6/faux/metrics"
	"github.com/influx6/gobuild/build"
	"github.com/influx6/moz/gen"
)

const (
	annotationFileFormat    = "%s_annotation_%s.%s"
	altAnnotationFileFormat = "%s_annotation_%s"
)

// Contains giving sets of variables exposing sytem GOPATH and GOPATHSRC.
var (
	goPath    = os.Getenv("GOPATH")
	goSrcPath = filepath.Join(goPath, "src")

	spaces     = regexp.MustCompile(`/s+`)
	itag       = regexp.MustCompile(`((\w+):"(\w+|[\w,?\s+\w]+)")`)
	annotation = regexp.MustCompile("@(\\w+(:\\w+)?)(\\([.\\s\\S]+\\))?")

	ASTTemplatFuncs = map[string]interface{}{
		"getTag":            GetTag,
		"fieldFor":          FieldFor,
		"getFields":         GetFields,
		"fieldNameFor":      FieldNameFor,
		"mapoutFields":      MapOutFields,
		"mapoutValues":      MapOutValues,
		"fieldByName":       FieldByFieldName,
		"randomValue":       RandomFieldAssign,
		"fieldsJSON":        MapOutFieldsToJSON,
		"randomValuesJSON":  MapOutFieldsWithRandomValuesToJSON,
		"stringValueFor":    ToValueString,
		"defaultValue":      AssignDefaultValue,
		"randomFieldValue":  RandomFieldValue,
		"defaultType":       DefaultTypeValueString,
		"defaultFieldValue": DefaultFieldValue,
	}

	naturalIdents = map[string]bool{
		"string":      true,
		"bool":        true,
		"rune":        true,
		"byte":        true,
		"int":         true,
		"int8":        true,
		"int16":       true,
		"int32":       true,
		"int64":       true,
		"uint":        true,
		"uint8":       true,
		"uint32":      true,
		"uint64":      true,
		"uintptr":     true,
		"float32":     true,
		"float64":     true,
		"complex128":  true,
		"complex64":   true,
		"error":       true,
		"struct":      true,
		"interface":   true,
		"interface{}": true,
		"struct{}":    true,
	}
)

// Packages defines a type to represent a slice of Packages.
type Packages []Package

// TestPackageForFile returns package associated with path.
func (pkgs Packages) TestPackageForFile(path string, targetFile string) (PackageDeclaration, Package, bool) {
	pl, ok := pkgs.TestPackageFor(path)
	if !ok {
		return PackageDeclaration{}, Package{}, ok
	}

	plDeclr, ok := pl.DeclarationFor(path, targetFile)
	return plDeclr, pl, ok
}

// PackageForFile returns package associated with path.
func (pkgs Packages) PackageForFile(path string, targetFile string) (PackageDeclaration, Package, bool) {
	pl, ok := pkgs.PackageFor(path)
	if !ok {
		return PackageDeclaration{}, Package{}, ok
	}

	plDeclr, ok := pl.DeclarationFor(path, targetFile)
	return plDeclr, pl, ok
}

// TestPackageFor returns package associated with path for its tests.
func (pkgs Packages) TestPackageFor(path string) (Package, bool) {
	for _, pkg := range pkgs {
		if pkg.Path == path && strings.HasSuffix(pkg.Tag, "_test") {
			return pkg, true
		}
	}

	return Package{}, false
}

// PackageFor returns package associated with path.
func (pkgs Packages) PackageFor(path string) (Package, bool) {
	for _, pkg := range pkgs {
		if pkg.Path == path && !strings.HasSuffix(pkg.Tag, "_test") {
			return pkg, true
		}
	}

	return Package{}, false
}

// ImportDeclaration defines a type to contain import declaration within a package.
type ImportDeclaration struct {
	Name        string
	Path        string
	Source      string
	Comments    string
	InternalPkg bool
}

// Package defines the central repository of all PackageDeclaration.
type Package struct {
	Name         string
	Tag          string
	Path         string
	FilePath     string
	Files        []string
	BuildPkg     *build.Package
	Packages     []PackageDeclaration
	TestPackages []PackageDeclaration
}

// Load calls all internal packages to load their respective imports.
func (pkg *Package) loadImported(m metrics.Metrics) error {
	for index, item := range pkg.Packages {
		if err := item.loadImported(m); err != nil {
			return err
		}

		pkg.Packages[index] = item
	}

	return nil
}

// HasFunctionFor returns true/false if the giving Struct Declaration has the giving function name.
func (pkg Package) HasFunctionFor(str StructDeclaration, funcName string) bool {
	for _, elem := range pkg.Packages {
		if elem.HasFunctionFor(str, funcName) {
			return true
		}
	}

	return false
}

// PackagesWithAnnotation returns a slice of all PackageDeclaration which have the annotation at package level.
func (pkg Package) PackagesWithAnnotation(name string) []PackageDeclaration {
	var pkgs []PackageDeclaration

	for _, elem := range pkg.Packages {
		if !elem.HasAnnotation(name) {
			continue
		}
		pkgs = append(pkgs, elem)
	}

	return pkgs
}

// AnnotationFirstFor returns all annotations with the giving name.
func (pkg Package) AnnotationFirstFor(typeName string) (AnnotationDeclaration, PackageDeclaration, bool) {
	for _, elem := range pkg.Packages {
		if annon, ok := elem.GetAnnotation(typeName); ok {
			return annon, elem, true
		}
	}

	return AnnotationDeclaration{}, PackageDeclaration{}, false
}

// HasAnnotation returns true/false if the giving package has any files having
// a giving annotation on the package level.
func (pkg Package) HasAnnotation(name string) bool {
	for _, elem := range pkg.Packages {
		if elem.HasAnnotation(name) {
			return true
		}
	}

	return false
}

// AnnotationsFor returns all annotations with the giving name.
func (pkg Package) AnnotationsFor(typeName string) []AnnotationDeclaration {
	var found []AnnotationDeclaration

	for _, elem := range pkg.Packages {
		found = append(found, elem.AnnotationsFor(typeName)...)
	}

	return found
}

// FunctionsForName returns a slice of FuncDeclaration for the giving name.
func (pkg Package) FunctionsForName(objName string) []FuncDeclaration {
	var funcs []FuncDeclaration

	for _, elem := range pkg.Packages {
		funcs = append(funcs, elem.FunctionsForName(objName)...)
	}

	return funcs
}

// ImportFor returns the ImportDeclaration associated with the giving handle.
// Returns error if the import is not found.
func (pkg Package) ImportFor(imp string) (ImportDeclaration, error) {
	for _, elem := range pkg.Packages {
		if impDeclr, err := elem.ImportFor(imp); err == nil {
			return impDeclr, nil
		}
	}

	return ImportDeclaration{}, errors.New("Not found")
}

// FunctionsFor returns a slice of FuncDeclaration for the giving object.
func (pkg Package) FunctionsFor(obj *ast.Object) []FuncDeclaration {
	var funcs []FuncDeclaration

	for _, elem := range pkg.Packages {
		funcs = append(funcs, elem.FunctionsFor(obj)...)
	}

	return funcs
}

// TestDeclarations returns the associated test declaration for the giving import path.
func (pkg Package) TestDeclarations(importPath string) []PackageDeclaration {
	var declrs []PackageDeclaration

	for _, declr := range pkg.TestPackages {
		if declr.Path == importPath {
			declrs = append(declrs, declr)
		}
	}

	return declrs
}

// Declarations returns the associated declaration for the giving import path.
func (pkg Package) Declarations(importPath string) []PackageDeclaration {
	var declrs []PackageDeclaration

	for _, declr := range pkg.Packages {
		if declr.Path == importPath {
			declrs = append(declrs, declr)
		}
	}

	return declrs
}

// TestDeclarationFor returns the associated test declaration for the giving file path.
func (pkg Package) TestDeclarationFor(importPath string, targetFile string) (PackageDeclaration, bool) {
	declrs := pkg.TestDeclarations(importPath)
	for _, declr := range declrs {
		if declr.File == targetFile {
			return declr, true
		}
	}

	return PackageDeclaration{}, false
}

// DeclarationFor returns the associated declaration for the giving file path.
func (pkg Package) DeclarationFor(importPath string, targetFile string) (PackageDeclaration, bool) {
	declrs := pkg.Declarations(importPath)
	for _, declr := range declrs {
		if declr.File == targetFile {
			return declr, true
		}
	}

	return PackageDeclaration{}, false
}

// TypeFor returns associated TypeDeclaration for importPath in file with the typeName.
func (pkg Package) TypeFor(importPath string, typeName string) (TypeDeclaration, bool) {
	for _, declr := range pkg.Declarations(importPath) {
		for _, elem := range declr.Types {
			if elem.Object.Name.Name == typeName {
				return elem, true
			}
		}
	}

	return TypeDeclaration{}, false
}

// FunctionFor returns associated FuncDeclaration for importPath in file with the typeName.
func (pkg Package) FunctionFor(importPath string, typeName string) (FuncDeclaration, bool) {
	for _, declr := range pkg.Declarations(importPath) {
		for _, elem := range declr.Functions {
			if elem.FuncDeclr.Name.Name == typeName {
				return elem, true
			}
		}
	}

	return FuncDeclaration{}, false
}

// StructFor returns associated StructDeclaration for importPath in file with the typeName.
func (pkg Package) StructFor(importPath string, typeName string) (StructDeclaration, bool) {
	for _, declr := range pkg.Declarations(importPath) {
		for _, elem := range declr.Structs {
			if elem.Object.Name.Name == typeName {
				return elem, true
			}
		}
	}

	return StructDeclaration{}, false
}

// InterfaceFor returns associated InterfaceDeclaration for importPath in file with the typeName.
func (pkg Package) InterfaceFor(importPath string, typeName string) (InterfaceDeclaration, bool) {
	for _, declr := range pkg.Declarations(importPath) {
		for _, elem := range declr.Interfaces {
			if elem.Object.Name.Name == typeName {
				return elem, true
			}
		}
	}

	return InterfaceDeclaration{}, false
}

// TypeForFile returns associated TypeDeclaration for importPath in file with the typeName.
func (pkg Package) TypeForFile(importPath string, targetFile string, typeName string) (TypeDeclaration, bool) {
	declr, ok := pkg.DeclarationFor(importPath, targetFile)
	if !ok {
		return TypeDeclaration{}, false
	}

	return declr.TypeFor(typeName)
}

// FunctionForFile returns associated FuncDeclaration for importPath in file with the typeName.
func (pkg Package) FunctionForFile(importPath string, targetFile string, typeName string) (FuncDeclaration, bool) {
	declr, ok := pkg.DeclarationFor(importPath, targetFile)
	if !ok {
		return FuncDeclaration{}, false
	}

	return declr.FunctionFor(typeName)
}

// StructForFile returns associated StructDeclaration for importPath in file with the typeName.
func (pkg Package) StructForFile(importPath string, targetFile string, typeName string) (StructDeclaration, bool) {
	declr, ok := pkg.DeclarationFor(importPath, targetFile)
	if !ok {
		return StructDeclaration{}, false
	}

	return declr.StructFor(typeName)
}

// InterfaceForFile returns associated InterfaceDeclaration for importPath in file with the typeName.
func (pkg Package) InterfaceForFile(importPath string, targetFile string, typeName string) (InterfaceDeclaration, bool) {
	declr, ok := pkg.DeclarationFor(importPath, targetFile)
	if !ok {
		return InterfaceDeclaration{}, false
	}

	return declr.InterfaceFor(typeName)
}

//===========================================================================================================

// PackageDeclaration defines a type which holds details relating to annotations declared on a
// giving package.
type PackageDeclaration struct {
	Package          string
	Path             string
	FilePath         string
	File             string
	Source           string
	Comments         []string
	Imports          map[string]ImportDeclaration
	Annotations      []AnnotationDeclaration
	Types            []TypeDeclaration
	Structs          []StructDeclaration
	Interfaces       []InterfaceDeclaration
	Functions        []FuncDeclaration
	Variables        []VariableDeclaration
	ObjectFunc       map[*ast.Object][]FuncDeclaration
	ImportedPackages map[string]Packages
	importedloaded   bool
}

// loadImported will attempt to load all available imported package that
// are not internal to go.
func (pkg *PackageDeclaration) loadImported(m metrics.Metrics) error {
	if pkg.importedloaded {
		return nil
	}

	pkg.importedloaded = true

	if pkg.ImportedPackages == nil {
		pkg.ImportedPackages = make(map[string]Packages)
	}

	for _, imported := range pkg.Imports {
		if imported.InternalPkg {
			continue
		}

		if _, ok := pkg.ImportedPackages[imported.Path]; ok {
			continue
		}

		importDir := filepath.Join(goSrcPath, imported.Path)
		uniqueImportDir := importDir + "#" + imported.Name
		processedPackages.pl.Lock()
		if res, ok := processedPackages.pkgs[uniqueImportDir]; ok {
			processedPackages.pl.Unlock()
			pkg.ImportedPackages[imported.Path] = Packages{res}
			continue
		}
		processedPackages.pl.Unlock()

		importedPkgs, err := PackageWithBuildCtx(m, importDir, build.Default)
		if err != nil {
			return err
		}

		pkg.ImportedPackages[imported.Path] = importedPkgs
	}

	return nil
}

// HasFunctionFor returns true/false if the giving Struct Declaration has the giving function name.
func (pkg PackageDeclaration) HasFunctionFor(str StructDeclaration, funcName string) bool {
	functions := Functions(pkg.FunctionsFor(str.Object.Name.Obj))

	if _, err := functions.Find(funcName); err != nil {
		return false
	}

	return true
}

// HasAnnotation returns true/false if giving PackageDeclaration has annotation at package level.
func (pkg PackageDeclaration) HasAnnotation(typeName string) bool {
	typeName = strings.TrimPrefix(typeName, "@")
	for _, item := range pkg.Annotations {
		if strings.TrimPrefix(item.Name, "@") != typeName {
			continue
		}

		return true
	}

	return false
}

// GetAnnotation returns the first annotation with the giving name.
func (pkg PackageDeclaration) GetAnnotation(typeName string) (AnnotationDeclaration, bool) {
	typeName = strings.TrimPrefix(typeName, "@")
	for _, item := range pkg.Annotations {
		if strings.TrimPrefix(item.Name, "@") != typeName {
			continue
		}
		return item, true
	}

	return AnnotationDeclaration{}, false
}

// AnnotationsFor returns all annotations with the giving name.
func (pkg PackageDeclaration) AnnotationsFor(typeName string) []AnnotationDeclaration {
	typeName = strings.TrimPrefix(typeName, "@")

	var found []AnnotationDeclaration

	for _, item := range pkg.Annotations {
		if strings.TrimPrefix(item.Name, "@") != typeName {
			continue
		}

		found = append(found, item)
	}

	return found
}

// FunctionsForName returns a slice of FuncDeclaration for the giving name.
func (pkg PackageDeclaration) FunctionsForName(objName string) []FuncDeclaration {
	var funcs []FuncDeclaration

	for obj, list := range pkg.ObjectFunc {
		if obj.Name != objName {
			continue
		}

		funcs = append(funcs, list...)
	}

	return funcs
}

// ImportFor returns the ImportDeclaration associated with the giving handle.
// Returns error if the import is not found.
func (pkg PackageDeclaration) ImportFor(imp string) (ImportDeclaration, error) {
	impDeclr, ok := pkg.Imports[imp]
	if !ok {
		return ImportDeclaration{}, errors.New("Not found")
	}

	return impDeclr, nil
}

// FunctionFor returns associated FuncDeclaration associated with name.
func (pkg PackageDeclaration) FunctionFor(typeName string) (FuncDeclaration, bool) {
	for _, typed := range pkg.Functions {
		if typed.FuncDeclr.Name.Name == typeName {
			return typed, true
		}
	}

	return FuncDeclaration{}, false
}

// TypeFor returns associated TypeDeclaration associated with name.
func (pkg PackageDeclaration) TypeFor(typeName string) (TypeDeclaration, bool) {
	for _, typed := range pkg.Types {
		if typed.Object.Name.Name == typeName {
			return typed, true
		}
	}

	return TypeDeclaration{}, false
}

// InterfaceFor returns associated InterfaceDeclaration associated with name.
func (pkg PackageDeclaration) InterfaceFor(intrName string) (InterfaceDeclaration, bool) {
	for _, inter := range pkg.Interfaces {
		if inter.Object.Name.Name == intrName {
			return inter, true
		}
	}

	return InterfaceDeclaration{}, false
}

// StructFor returns associated StructDeclaration associated with name.
func (pkg PackageDeclaration) StructFor(structName string) (StructDeclaration, bool) {
	for _, structd := range pkg.Structs {
		if structd.Object.Name.Name == structName {
			return structd, true
		}
	}

	return StructDeclaration{}, false
}

// FunctionsFor returns a slice of FuncDeclaration for the giving object.
func (pkg PackageDeclaration) FunctionsFor(obj *ast.Object) []FuncDeclaration {
	if funcs, ok := pkg.ObjectFunc[obj]; ok {
		return funcs
	}

	return pkg.FunctionsForName(obj.Name)
}

//===========================================================================================================

// VariableDeclaration defines a type which holds annotation data for a giving variable declaration.
type VariableDeclaration struct {
	From         int
	Length       int
	Package      string
	Path         string
	FilePath     string
	Source       string
	Comments     string
	File         string
	Position     token.Pos
	Object       *ast.ValueSpec
	GenObj       *ast.GenDecl
	Declr        *PackageDeclaration
	Annotations  []AnnotationDeclaration
	Associations map[string]AnnotationAssociationDeclaration
}

// StructDeclaration defines a type which holds annotation data for a giving struct type declaration.
type StructDeclaration struct {
	From         int
	Length       int
	Package      string
	Path         string
	FilePath     string
	Source       string
	Comments     string
	File         string
	Struct       *ast.StructType
	Object       *ast.TypeSpec
	GenObj       *ast.GenDecl
	Position     token.Pos
	Declr        *PackageDeclaration
	Annotations  []AnnotationDeclaration
	Associations map[string]AnnotationAssociationDeclaration
}

// AnnotationsFor returns all annotations with the giving name.
func (str StructDeclaration) AnnotationsFor(typeName string) []AnnotationDeclaration {
	typeName = strings.TrimPrefix(typeName, "@")

	var found []AnnotationDeclaration

	for _, item := range str.Annotations {
		if strings.TrimPrefix(item.Name, "@") != typeName {
			continue
		}

		found = append(found, item)
	}

	return found
}

// TypeDeclaration defines a type which holds annotation data for a giving type declaration.
type TypeDeclaration struct {
	From         int
	Length       int
	Package      string
	Path         string
	FilePath     string
	Source       string
	Comments     string
	File         string
	Object       *ast.TypeSpec
	GenObj       *ast.GenDecl
	Position     token.Pos
	Declr        *PackageDeclaration
	Annotations  []AnnotationDeclaration
	Associations map[string]AnnotationAssociationDeclaration
}

// AnnotationsFor returns all annotations with the giving name.
func (ty TypeDeclaration) AnnotationsFor(typeName string) []AnnotationDeclaration {
	typeName = strings.TrimPrefix(typeName, "@")

	var found []AnnotationDeclaration

	for _, item := range ty.Annotations {
		if strings.TrimPrefix(item.Name, "@") != typeName {
			continue
		}

		found = append(found, item)
	}

	return found
}

//===========================================================================================================

// FuncDeclaration defines a type used to annotate a giving type declaration
// associated with a ast for a function.
type FuncDeclaration struct {
	From            int
	Length          int
	Package         string
	Path            string
	FilePath        string
	Exported        bool
	File            string
	FuncName        string
	RecieverName    string
	Source          string
	Comments        string
	Position        token.Pos
	TypeDeclr       ast.Decl
	FuncDeclr       *ast.FuncDecl
	Type            *ast.FuncType
	Reciever        *ast.Object
	RecieverIdent   *ast.Ident
	RecieverPointer *ast.StarExpr
	FuncType        *ast.FieldList
	Returns         *ast.FieldList
	Arguments       *ast.FieldList
	Declr           *PackageDeclaration
	Annotations     []AnnotationDeclaration
	Associations    map[string]AnnotationAssociationDeclaration
}

// AnnotationsFor returns all annotations with the giving name.
func (fun FuncDeclaration) AnnotationsFor(typeName string) []AnnotationDeclaration {
	typeName = strings.TrimPrefix(typeName, "@")

	var found []AnnotationDeclaration

	for _, item := range fun.Annotations {
		if strings.TrimPrefix(item.Name, "@") != typeName {
			continue
		}

		found = append(found, item)
	}

	return found
}

// GetAnnotation returns AnnotationDeclaration if giving FuncDeclaration has annotation at package level.
func (fun FuncDeclaration) GetAnnotation(typeName string) (AnnotationDeclaration, bool) {
	typeName = strings.TrimPrefix(typeName, "@")
	for _, item := range fun.Annotations {
		if strings.TrimPrefix(item.Name, "@") != typeName {
			continue
		}

		return item, true
	}

	return AnnotationDeclaration{}, false
}

// HasAnnotation returns true/false if giving FuncDeclaration has annotation at package level.
func (fun FuncDeclaration) HasAnnotation(typeName string) bool {
	typeName = strings.TrimPrefix(typeName, "@")
	for _, item := range fun.Annotations {
		if strings.TrimPrefix(item.Name, "@") != typeName {
			continue
		}

		return true
	}

	return false
}

// Definition returns a FunctionDefinition for this function.
func (fun FuncDeclaration) Definition(pkg *PackageDeclaration) (FunctionDefinition, error) {
	return GetFunctionDefinitionFromDeclaration(fun, pkg)
}

// Functions defines a slice of FuncDeclaration.
type Functions []FuncDeclaration

// Find returns the giving Function of the giving name.
func (fnList Functions) Find(name string) (FuncDeclaration, error) {
	for _, fn := range fnList {
		if fn.FuncName == name {
			return fn, nil
		}
	}

	return FuncDeclaration{}, fmt.Errorf("Function with %q not found", name)
}

//===========================================================================================================

// AnnotationAssociationDeclaration defines a type which defines an association between
// a giving annotation and a series of values.
type AnnotationAssociationDeclaration struct {
	Annotation string
	Action     string
	Template   string
	TypeName   string
	Record     AnnotationDeclaration
}

// InterfaceDeclaration defines a type which holds annotation data for a giving interface type declaration.
type InterfaceDeclaration struct {
	From         int
	Length       int
	Package      string
	Path         string
	Source       string
	Comments     string
	FilePath     string
	File         string
	Interface    *ast.InterfaceType
	Object       *ast.TypeSpec
	GenObj       *ast.GenDecl
	Position     token.Pos
	Declr        *PackageDeclaration
	Annotations  []AnnotationDeclaration
	Associations map[string]AnnotationAssociationDeclaration
}

// Methods returns the associated methods for the giving interface.
func (i InterfaceDeclaration) Methods(pkg *PackageDeclaration) []FunctionDefinition {
	return GetInterfaceFunctions(i.Interface, pkg)
}

// ArgType defines a type to represent the information for a giving functions argument or
// return type declaration.
type ArgType struct {
	Name            string
	Type            string
	ExType          string
	Package         string
	BaseType        bool
	Import          ImportDeclaration
	Import2         ImportDeclaration
	NameObject      *ast.Object
	TypeObject      *ast.Object
	StructObject    *ast.StructType
	Spec            *ast.TypeSpec
	InterfaceObject *ast.InterfaceType
	ImportedObject  *ast.SelectorExpr
	SelectPackage   *ast.Ident
	SelectObject    *ast.Ident
	ArrayType       *ast.ArrayType
	MapType         *ast.MapType
	ChanType        *ast.ChanType
	PointerType     *ast.StarExpr
	IdentType       *ast.Ident
	Tags            []TagDeclaration
}

// FunctionDefinition defines a type to represent the function/method declarations of an
// interface type.
type FunctionDefinition struct {
	Name      string
	Args      []ArgType
	Returns   []ArgType
	Func      *ast.FuncType
	Interface *ast.InterfaceType
	Struct    *ast.StructType
}

// ArgumentNamesList returns the assignment names for the function arguments.
func (fd FunctionDefinition) ArgumentNamesList() string {
	var args []string

	for _, arg := range fd.Args {
		args = append(args, fmt.Sprintf("%s", arg.Name))
	}

	return strings.Join(args, ",")
}

// ReturnNamesList returns the assignment names for the return arguments
func (fd FunctionDefinition) ReturnNamesList() string {
	var rets []string

	for _, ret := range fd.Returns {
		rets = append(rets, fmt.Sprintf("%s", ret.Name))
	}

	return strings.Join(rets, ",")
}

// ReturnList returns a string version of the return of the giving function.
func (fd FunctionDefinition) ReturnList(asFromOutside bool) string {
	var rets []string

	for _, ret := range fd.Returns {
		if asFromOutside {
			rets = append(rets, fmt.Sprintf("%s", ret.ExType))
			continue
		}

		rets = append(rets, fmt.Sprintf("%s", ret.Type))
	}

	return strings.Join(rets, ",")
}

// ArgumentList returns a string version of the arguments of the giving function.
func (fd FunctionDefinition) ArgumentList(asFromOutside bool) string {
	var args []string

	for _, arg := range fd.Args {
		if asFromOutside {
			args = append(args, fmt.Sprintf("%s %s", arg.Name, arg.ExType))
			continue
		}

		args = append(args, fmt.Sprintf("%s %s", arg.Name, arg.Type))
	}

	return strings.Join(args, ",")
}

//===========================================================================================================

// Fields defines a slice type of FieldDeclaration.
type Fields []FieldDeclaration

// Normal defines a function that returns all fields which are non-embedded.
func (flds Fields) Normal() Fields {
	var fields Fields

	for _, declr := range flds {
		if declr.Embedded {
			continue
		}

		fields = append(fields, declr)
	}

	return fields
}

// Embedded defines a function that returns all appropriate Field
// that match the giving tagName
func (flds Fields) Embedded() Fields {
	var fields Fields

	for _, declr := range flds {
		if declr.Embedded {
			fields = append(fields, declr)
		}
	}

	return fields
}

// TagFor defines a function that returns all appropriate TagDeclaration
// that match the giving tagName
func (flds Fields) TagFor(tagName string) []TagDeclaration {
	var declrs []TagDeclaration

	for _, declr := range flds {
		if dl, err := declr.GetTag(tagName); err == nil {
			declrs = append(declrs, dl)
		}
	}

	return declrs
}

// FieldDeclaration defines a type to represent a giving struct fields and tags.
type FieldDeclaration struct {
	Exported      bool
	Embedded      bool
	IsStruct      bool
	FieldName     string
	FieldTypeName string
	Field         *ast.Field
	Type          *ast.Object
	Spec          *ast.TypeSpec
	Struct        *ast.StructType
	Tags          []TagDeclaration
	Arg           ArgType
}

// GetFields returns all fields associated with the giving struct but skips
func GetFields(str StructDeclaration, pkg *PackageDeclaration) []FieldDeclaration {
	var fields []FieldDeclaration

	var counter int
	for _, item := range str.Struct.Fields.List {
		counter++
		arg, err := GetArgTypeFromField(counter, "var", pkg.File, item, pkg)
		if err != nil {
			continue
		}

		var field FieldDeclaration
		field.Arg = arg
		field.Type = arg.TypeObject
		field.Spec = arg.Spec
		field.Struct = arg.StructObject
		field.Field = item
		field.FieldName = arg.Name
		field.FieldTypeName = arg.Type

		if len(item.Names) == 0 {
			field.Exported = true
			field.Embedded = true
		}

		if arg.Name != strings.ToLower(arg.Name) {
			field.Exported = true
		}

		for _, tag := range arg.Tags {
			tag.Field = field
			field.Tags = append(field.Tags, tag)
		}

		fields = append(fields, field)
	}

	return fields
}

// GetTag returns the giving tag associated with the name if it exists.
func (f FieldDeclaration) GetTag(tagName string) (TagDeclaration, error) {
	for _, tag := range f.Tags {
		if tag.Name == tagName {
			return tag, nil
		}
	}

	return TagDeclaration{}, fmt.Errorf("Tag for %q not found", tagName)
}

// TagDeclaration defines a type which represents a giving tag declaration for a provided type.
type TagDeclaration struct {
	Name  string
	Value string
	Metas []string
	Base  string
	Field FieldDeclaration
}

// Has returns true/false if the tag.Metas has the given value in the list.
func (t TagDeclaration) Has(item string) bool {
	for _, meta := range t.Metas {
		if meta == item {
			return true
		}
	}

	return false
}

// ToValueString returns the string representation of a basic go core datatype.
func ToValueString(val interface{}) string {
	switch bo := val.(type) {
	case *time.Time:
		return bo.UTC().String()
	case time.Time:
		return bo.UTC().String()
	case string:
		return strconv.Quote(bo)
	case int:
		return strconv.Itoa(bo)
	case int64:
		return strconv.Itoa(int(bo))
	case rune:
		return strconv.QuoteRune(bo)
	case bool:
		return strconv.FormatBool(bo)
	case byte:
		return strconv.QuoteRune(rune(bo))
	case float64:
		return strconv.FormatFloat(bo, 'f', 4, 64)
	case float32:
		return strconv.FormatFloat(float64(bo), 'f', 4, 64)
	default:
		data, err := json.Marshal(val)
		if err != nil {
			return err.Error()
		}

		return string(data)
	}
}

//===========================================================================================================

// GetIdentName returns the first indent found within the field if it exists.
func GetIdentName(field *ast.Field) (*ast.Ident, error) {
	if len(field.Names) == 0 {
		return nil, ErrEmptyList
	}

	return field.Names[0], nil
}

// GetArgTypeFromField returns a ArgType that writes out the representation of the giving variable name or decleration ast.Field
// associated with the giving package. It returns an error if it does not know the type.
func GetArgTypeFromField(retCounter int, varPrefix string, targetFile string, result *ast.Field, pkg *PackageDeclaration) (ArgType, error) {
	var tags []TagDeclaration

	if result.Tag != nil {
		tagList := strings.Split(spaces.ReplaceAllString(result.Tag.Value, " "), " ")
		for _, tag := range tagList {
			if !itag.MatchString(tag) {
				continue
			}

			res := itag.FindStringSubmatch(tag)
			resValue := strings.Split(res[3], ",")

			tags = append(tags, TagDeclaration{
				Base:  res[0],
				Name:  res[2],
				Value: resValue[0],
				Metas: resValue[1:],
			})
		}
	}

	resPkg, defaultresType := getPackageFromItem(result.Type, filepath.Base(pkg.Package))

	switch iobj := result.Type.(type) {
	case *ast.InterfaceType:
		var nameObj *ast.Object

		var name string
		resName, err := GetIdentName(result)
		switch err != nil {
		case true:
			name = fmt.Sprintf("%s%d", varPrefix, retCounter)
		case false:
			name = resName.Name
			nameObj = resName.Obj
		}

		arg := ArgType{
			Name:            name,
			Tags:            tags,
			NameObject:      nameObj,
			Type:            getName(iobj),
			InterfaceObject: iobj,
			Package:         resPkg,
			BaseType:        defaultresType,
			ExType:          getNameAsFromOuter(iobj, filepath.Base(pkg.Package)),
		}
		return arg, nil

	case *ast.Ident:

		var nameObj *ast.Object

		var name string
		resName, err := GetIdentName(result)
		switch err != nil {
		case true:
			name = fmt.Sprintf("%s%d", varPrefix, retCounter)
		case false:
			name = resName.Name
			nameObj = resName.Obj
		}

		arg := ArgType{
			Name:       name,
			Tags:       tags,
			NameObject: nameObj,
			Type:       getName(iobj),
			ExType:     getNameAsFromOuter(iobj, filepath.Base(pkg.Package)),
			TypeObject: iobj.Obj,
			Package:    resPkg,
			BaseType:   defaultresType,
		}

		if iobj.Obj != nil && iobj.Obj.Decl != nil {
			if def, ok := iobj.Obj.Decl.(*ast.TypeSpec); ok {
				arg.Spec = def
				switch obx := def.Type.(type) {
				case *ast.StructType:
					arg.StructObject = obx
				case *ast.InterfaceType:
					arg.InterfaceObject = obx
				}
			}
		}

		return arg, nil

	case *ast.SelectorExpr:

		xobj, ok := iobj.X.(*ast.Ident)
		if !ok {
			return ArgType{}, errors.New("Saw ast.SelectorExpr but X is not an *ast.Ident type")
		}

		importDclr, err := pkg.ImportFor(xobj.Name)
		if err != nil {
			return ArgType{}, err
		}

		var name string
		resName, err := GetIdentName(result)
		switch err != nil {
		case true:
			name = fmt.Sprintf("%s%d", varPrefix, retCounter)
		case false:
			name = resName.Name
		}

		arg := ArgType{
			Name:           name,
			Import:         importDclr,
			Tags:           tags,
			Package:        xobj.Name,
			ImportedObject: iobj,
			Type:           getName(iobj),
			ExType:         getNameAsFromOuter(iobj, filepath.Base(pkg.Package)),
		}

		arg.SelectPackage = xobj
		arg.SelectObject = iobj.Sel

		if !importDclr.InternalPkg {
			importedParentPackage, ok := pkg.ImportedPackages[importDclr.Path]
			if !ok {
				return ArgType{}, fmt.Errorf("Expected to have loaded imported package %q with tag %q", importDclr.Path, importDclr.Name)
			}

			mdeclr, ok := importedParentPackage.PackageFor(importDclr.Path)
			if !ok {
				return ArgType{}, fmt.Errorf("Expected to have found moz.Package for %q with tag %q", importDclr.Path, importDclr.Name)
			}

			if mtype, ok := mdeclr.TypeFor(importDclr.Path, iobj.Sel.Name); ok {
				arg.Spec = mtype.Object
			}

			if stype, ok := mdeclr.StructFor(importDclr.Path, iobj.Sel.Name); ok {
				arg.Spec = stype.Object
				arg.StructObject = stype.Struct
			}

			if itype, ok := mdeclr.InterfaceFor(importDclr.Path, iobj.Sel.Name); ok {
				arg.Spec = itype.Object
				arg.InterfaceObject = itype.Interface
			}
		}

		return arg, nil

	case *ast.StarExpr:

		var name string
		resName, err := GetIdentName(result)
		switch err != nil {
		case true:
			name = fmt.Sprintf("%s%d", varPrefix, retCounter)
		case false:
			name = resName.Name
		}

		var arg ArgType
		arg.Tags = tags
		arg.Name = name
		arg.PointerType = iobj
		arg.Type = getName(iobj)
		arg.ExType = getNameAsFromOuter(iobj, filepath.Base(pkg.Package))

		switch value := iobj.X.(type) {
		case *ast.SelectorExpr:
			arg.ImportedObject = value

			vob, ok := value.X.(*ast.Ident)
			if !ok {
				return ArgType{}, errors.New("Saw ast.SelectorExpr but X is not an *ast.Ident type")
			}

			importDclr, err := pkg.ImportFor(vob.Name)
			if err != nil {
				return ArgType{}, err
			}

			arg.Package = vob.Name
			arg.Import = importDclr

			arg.SelectPackage = vob
			arg.SelectObject = value.Sel

			if !importDclr.InternalPkg {
				importedParentPackage, ok := pkg.ImportedPackages[importDclr.Path]
				if !ok {
					return ArgType{}, fmt.Errorf("Expected to have loaded imported package %q with tag %q", importDclr.Path, importDclr.Name)
				}

				mdeclr, ok := importedParentPackage.PackageFor(importDclr.Path)
				if !ok {
					return ArgType{}, fmt.Errorf("Expected to have found moz.Package for %q with tag %q", importDclr.Path, importDclr.Name)
				}

				if mtype, ok := mdeclr.TypeFor(importDclr.Path, value.Sel.Name); ok {
					arg.Spec = mtype.Object
				}

				if stype, ok := mdeclr.StructFor(importDclr.Path, value.Sel.Name); ok {
					arg.Spec = stype.Object
					arg.StructObject = stype.Struct
				}

				if itype, ok := mdeclr.InterfaceFor(importDclr.Path, value.Sel.Name); ok {
					arg.Spec = itype.Object
					arg.InterfaceObject = itype.Interface
				}
			}
		case *ast.InterfaceType:
			arg.InterfaceObject = value
		case *ast.StructType:
			arg.StructObject = value
		case *ast.ArrayType:
			arg.ArrayType = value
		case *ast.Ident:
			arg.IdentType = value
			arg.NameObject = value.Obj

			arg.Package = resPkg
			arg.BaseType = defaultresType

			if value.Obj != nil && value.Obj.Decl != nil {
				if def, ok := value.Obj.Decl.(*ast.TypeSpec); ok {
					arg.Spec = def
					switch obx := def.Type.(type) {
					case *ast.StructType:
						arg.StructObject = obx
					case *ast.InterfaceType:
						arg.InterfaceObject = obx
					}
				}
			}
		case *ast.ChanType:
			arg.ChanType = value
		}

		return arg, nil

	case *ast.MapType:

		var name string
		resName, err := GetIdentName(result)
		switch err != nil {
		case true:
			name = fmt.Sprintf("%s%d", varPrefix, retCounter)
		case false:
			name = resName.Name
		}

		var arg ArgType
		arg.Name = name
		arg.Tags = tags
		arg.MapType = iobj
		arg.Type = getName(iobj)
		arg.ExType = getNameAsFromOuter(iobj, filepath.Base(pkg.Package))

		if keySel, err := getSelector(iobj.Key); err == nil {
			if x, ok := keySel.X.(*ast.Ident); ok {
				if imported, err := pkg.ImportFor(x.Name); err == nil {
					arg.Import = imported
				}
			}
		}

		if valSel, err := getSelector(iobj.Value); err == nil {
			if x, ok := valSel.X.(*ast.Ident); ok {
				if imported, err := pkg.ImportFor(x.Name); err == nil {
					arg.Import2 = imported
				}
			}
		}

		return arg, nil
	case *ast.ArrayType:

		var name string
		resName, err := GetIdentName(result)
		switch err != nil {
		case true:
			name = fmt.Sprintf("%s%d", varPrefix, retCounter)
		case false:
			name = resName.Name
		}

		var arg ArgType
		arg.Name = name
		arg.Tags = tags
		arg.ArrayType = iobj
		arg.Type = getName(iobj)
		arg.ExType = getNameAsFromOuter(iobj, filepath.Base(pkg.Package))

		switch value := iobj.Elt.(type) {
		case *ast.SelectorExpr:
			arg.ImportedObject = value

			vob, ok := value.X.(*ast.Ident)
			if !ok {
				return ArgType{}, errors.New("Saw ast.SelectorExpr but X is not an *ast.Ident type")
			}

			importDclr, err := pkg.ImportFor(vob.Name)
			if err != nil {
				return ArgType{}, err
			}

			arg.Package = vob.Name
			arg.Import = importDclr

			arg.SelectPackage = vob
			arg.SelectObject = value.Sel

			if !importDclr.InternalPkg {
				importedParentPackage, ok := pkg.ImportedPackages[importDclr.Path]
				if !ok {
					return ArgType{}, fmt.Errorf("Expected to have loaded imported package %q with tag %q", importDclr.Path, importDclr.Name)
				}

				mdeclr, ok := importedParentPackage.PackageFor(importDclr.Path)
				if !ok {
					return ArgType{}, fmt.Errorf("Expected to have found moz.Package for %q with tag %q", importDclr.Path, importDclr.Name)
				}

				if mtype, ok := mdeclr.TypeFor(importDclr.Path, value.Sel.Name); ok {
					arg.Spec = mtype.Object
				}

				if stype, ok := mdeclr.StructFor(importDclr.Path, value.Sel.Name); ok {
					arg.Spec = stype.Object
					arg.StructObject = stype.Struct
				}

				if itype, ok := mdeclr.InterfaceFor(importDclr.Path, value.Sel.Name); ok {
					arg.Spec = itype.Object
					arg.InterfaceObject = itype.Interface
				}
			}
		case *ast.StarExpr:
			arg.PointerType = value
		case *ast.InterfaceType:
			arg.InterfaceObject = value
		case *ast.StructType:
			arg.StructObject = value
		case *ast.Ident:
			arg.IdentType = value
			arg.NameObject = value.Obj
			arg.Package = resPkg
			arg.BaseType = defaultresType
			if value.Obj != nil && value.Obj.Decl != nil {
				if def, ok := value.Obj.Decl.(*ast.TypeSpec); ok {
					arg.Spec = def
					switch obx := def.Type.(type) {
					case *ast.StructType:
						arg.StructObject = obx
					case *ast.InterfaceType:
						arg.InterfaceObject = obx
					}
				}
			}
		case *ast.ChanType:
			arg.ChanType = value
		}

		return arg, nil

	case *ast.ChanType:

		var name string
		resName, err := GetIdentName(result)
		switch err != nil {
		case true:
			name = fmt.Sprintf("%s%d", varPrefix, retCounter)
		case false:
			name = resName.Name
		}

		var arg ArgType
		arg.Name = name
		arg.Tags = tags
		arg.Type = getName(iobj.Value)
		arg.ExType = getNameAsFromOuter(iobj, filepath.Base(pkg.Package))

		switch value := iobj.Value.(type) {
		case *ast.SelectorExpr:
			arg.ImportedObject = value

			vob, ok := value.X.(*ast.Ident)
			if !ok {
				return ArgType{}, errors.New("Saw ast.SelectorExpr but X is not an *ast.Ident type")
			}

			importDclr, err := pkg.ImportFor(vob.Name)
			if err != nil {
				return ArgType{}, err
			}

			arg.Package = vob.Name
			arg.Import = importDclr
		case *ast.StarExpr:
			arg.PointerType = value
		case *ast.InterfaceType:
			arg.InterfaceObject = value
		case *ast.StructType:
			arg.StructObject = value
		case *ast.ArrayType:
			arg.ArrayType = value
		case *ast.Ident:
			arg.IdentType = value
			arg.NameObject = value.Obj

			arg.Package = resPkg
			arg.BaseType = defaultresType
			if value.Obj != nil && value.Obj.Decl != nil {
				if def, ok := value.Obj.Decl.(*ast.TypeSpec); ok {
					arg.Spec = def
					switch obx := def.Type.(type) {
					case *ast.StructType:
						arg.StructObject = obx
					case *ast.InterfaceType:
						arg.InterfaceObject = obx
					}
				}
			}
		case *ast.ChanType:
			arg.ChanType = value
		}

		return arg, nil
	}

	return ArgType{}, errors.New("Unknown Field type, only variable type declaration wanted")
}

// GetFunctionDefinitionFromField returns a FunctionDefinition representing a giving function.
func GetFunctionDefinitionFromField(method *ast.Field, pkg *PackageDeclaration) (FunctionDefinition, error) {
	if len(method.Names) == 0 {
		return FunctionDefinition{}, errors.New("Method field must have names")
	}

	nameIdent := method.Names[0]
	ftype, ok := method.Type.(*ast.FuncType)
	if !ok {
		return FunctionDefinition{}, errors.New("Only ast.FuncType allowed")
	}

	var arguments, returns []ArgType

	if ftype.Results != nil {
		var retCounter int
		for _, result := range ftype.Results.List {
			retCounter++
			arg, err := GetArgTypeFromField(retCounter, "ret", pkg.File, result, pkg)
			if err != nil {
				return FunctionDefinition{}, err
			}

			returns = append(returns, arg)
		}
	}

	if ftype.Params != nil {
		var varCounter int
		for _, param := range ftype.Params.List {
			varCounter++
			arg, err := GetArgTypeFromField(varCounter, "var", pkg.File, param, pkg)
			if err != nil {
				return FunctionDefinition{}, err
			}
			arguments = append(arguments, arg)
		}
	}

	return FunctionDefinition{
		Func:    ftype,
		Returns: returns,
		Args:    arguments,
		Name:    nameIdent.Name,
	}, nil
}

// GetFunctionDefinitionFromDeclaration returns a FunctionDefinition withe the associated FuncDeclaration.
func GetFunctionDefinitionFromDeclaration(funcObj FuncDeclaration, pkg *PackageDeclaration) (FunctionDefinition, error) {
	var arguments, returns []ArgType

	if funcObj.Type.Results != nil {
		var retCounter int
		for _, result := range funcObj.Type.Results.List {
			retCounter++
			arg, err := GetArgTypeFromField(retCounter, "ret", funcObj.File, result, pkg)
			if err != nil {
				return FunctionDefinition{}, err
			}

			returns = append(returns, arg)

		}
	}

	if funcObj.Type.Params != nil {
		var varCounter int
		for _, param := range funcObj.Type.Params.List {
			varCounter++
			arg, err := GetArgTypeFromField(varCounter, "var", funcObj.File, param, pkg)
			if err != nil {
				return FunctionDefinition{}, err
			}

			arguments = append(arguments, arg)
		}
	}

	var defs FunctionDefinition
	defs.Func = funcObj.Type
	defs.Returns = returns
	defs.Args = arguments
	defs.Name = funcObj.FuncName

	return defs, nil
}

// GetInterfaceFunctions returns a slice of FunctionDefinitions retrieved from the provided
// interface type object.
func GetInterfaceFunctions(intr *ast.InterfaceType, pkg *PackageDeclaration) []FunctionDefinition {
	var defs []FunctionDefinition

	for _, method := range intr.Methods.List {
		if len(method.Names) != 0 {
			if def, err := GetFunctionDefinitionFromField(method, pkg); err == nil {
				def.Interface = intr
				defs = append(defs, def)
			}
			continue
		}

		ident, ok := method.Type.(*ast.Ident)
		if !ok {
			continue
		}

		if ident == nil || ident.Obj == nil || ident.Obj.Decl == nil {
			continue
		}

		identDecl, ok := ident.Obj.Decl.(*ast.TypeSpec)
		if !ok {
			continue
		}

		identIntr, ok := identDecl.Type.(*ast.InterfaceType)
		if !ok {
			continue
		}

		defs = append(defs, GetInterfaceFunctions(identIntr, pkg)...)
	}

	return defs
}

// GetVariableTypeName returns a variable type name as exported from the base package
// returning the string representation.
func GetVariableTypeName(item interface{}) (string, error) {
	nameDeclr := getName(item)
	if nameDeclr == "" {
		return "", errors.New("Unknown type")
	}

	return nameDeclr, nil
}

// GetVariableNameAsExported returns a variable type name as exported from the base package
// returning the string representation.
func GetVariableNameAsExported(item interface{}, basePkg string) (string, error) {
	nameDeclr := getNameAsFromOuter(item, basePkg)
	if nameDeclr == "" {
		return "", errors.New("Unknown type")
	}

	return nameDeclr, nil
}

// getPackageFromItem returns the package name associated with the type
// by attempting to retrieve it from a selector or final declaration name,
// and returns true/false if its part of go's base types.
func getPackageFromItem(item interface{}, defaultPkg string) (string, bool) {
	realName := getRealIdentName(item)

	if parts := strings.Split(realName, "."); len(parts) > 1 {
		if _, ok := naturalIdents[parts[1]]; ok {
			return "", true
		}

		return parts[0], false
	}

	if _, ok := naturalIdents[realName]; ok {
		return "", true
	}

	return defaultPkg, false
}

func getSelector(item interface{}) (*ast.SelectorExpr, error) {
	switch di := item.(type) {
	case *ast.StarExpr:
		return getSelector(di.X)
	case *ast.ArrayType:
		return getSelector(di.Elt)
	case *ast.ChanType:
		return getSelector(di.Value)
	case *ast.SelectorExpr:
		return di, nil
	default:
		return nil, errors.New("Has no selector")
	}
}

func getRealIdentName(item interface{}) string {
	switch di := item.(type) {
	case *ast.StarExpr:
		return getRealIdentName(di.X)
	case *ast.SelectorExpr:
		xobj, ok := di.X.(*ast.Ident)
		if !ok {
			return ""
		}

		return fmt.Sprintf("%s.%s", xobj.Name, di.Sel.Name)
	case *ast.Ident:
		return di.Name
	case *ast.ArrayType:
		return getRealIdentName(di.Elt)
	case *ast.ChanType:
		return getRealIdentName(di.Value)
	default:
		return ""
	}
}

func getNameAsFromOuter(item interface{}, basePkg string) string {
	switch di := item.(type) {
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.MapType:
		keyName := getNameAsFromOuter(di.Key, basePkg)
		valName := getNameAsFromOuter(di.Value, basePkg)
		return fmt.Sprintf("map[%s]%s", keyName, valName)
	case *ast.StarExpr:
		return fmt.Sprintf("*%s", getNameAsFromOuter(di.X, basePkg))
	case *ast.SelectorExpr:
		xobj, ok := di.X.(*ast.Ident)
		if !ok {
			return ""
		}

		return fmt.Sprintf("%s.%s", xobj.Name, di.Sel.Name)
	case *ast.StructType:
		return "struct{}"
	case *ast.Ident:
		if _, ok := naturalIdents[di.Name]; ok {
			return di.Name
		}

		return fmt.Sprintf("%s.%s", basePkg, di.Name)
	case *ast.ArrayType:
		if di.Len != nil {
			if dlen, ok := di.Len.(*ast.Ident); ok {
				return fmt.Sprintf("[%s]%s", dlen.Name, getNameAsFromOuter(di.Elt, basePkg))
			}

			if dlen, ok := di.Len.(*ast.BasicLit); ok {
				return fmt.Sprintf("[%s]%s", dlen.Value, getNameAsFromOuter(di.Elt, basePkg))
			}
		}

		return fmt.Sprintf("[]%s", getNameAsFromOuter(di.Elt, basePkg))
	case *ast.ChanType:
		return fmt.Sprintf("chan %s", getNameAsFromOuter(di.Value, basePkg))
	default:
		return ""
	}
}

func getName(item interface{}) string {
	switch di := item.(type) {
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.MapType:
		// fmt.Printf("MapType: %#v : %#v\n", di.Key, di.Value)
		keyName := getName(di.Key)
		valName := getName(di.Value)
		return fmt.Sprintf("map[%s]%s", keyName, valName)
	case *ast.StarExpr:
		return fmt.Sprintf("*%s", getName(di.X))
	case *ast.SelectorExpr:
		xobj, ok := di.X.(*ast.Ident)
		if !ok {
			return ""
		}

		return fmt.Sprintf("%s.%s", xobj.Name, di.Sel.Name)
	case *ast.StructType:
		return "struct{}"
	case *ast.Ident:
		return di.Name
	case *ast.ArrayType:
		if di.Len != nil {
			if dlen, ok := di.Len.(*ast.Ident); ok {
				return fmt.Sprintf("[%s]%s", dlen.Name, getName(di.Elt))
			}

			if dlen, ok := di.Len.(*ast.BasicLit); ok {
				return fmt.Sprintf("[%s]%s", dlen.Value, getName(di.Elt))
			}
		}

		return fmt.Sprintf("[]%s", getName(di.Elt))
	case *ast.ChanType:
		return fmt.Sprintf("chan %s", getName(di.Value))
	default:
		return ""
	}
}

// FindStructType defines a function to search a package declaration Structs of a giving typeName.
func FindStructType(pkg PackageDeclaration, typeName string) (StructDeclaration, error) {
	for _, elem := range pkg.Structs {
		if elem.Object.Name.Name == typeName {
			return elem, nil
		}
	}

	return StructDeclaration{}, fmt.Errorf("Struct of type %q not found", typeName)
}

// FindInterfaceType defines a function to search a package declaration Interface of a giving typeName.
func FindInterfaceType(pkg PackageDeclaration, typeName string) (InterfaceDeclaration, error) {
	for _, elem := range pkg.Interfaces {
		if elem.Object.Name.Name == typeName {
			return elem, nil
		}
	}

	return InterfaceDeclaration{}, fmt.Errorf("Interface of type %q not found", typeName)
}

// FindType defines a function to search a package declaration Structs of a giving typeName.
func FindType(pkg PackageDeclaration, typeName string) (TypeDeclaration, error) {
	for _, elem := range pkg.Types {
		if elem.Object.Name.Name == typeName {
			return elem, nil
		}
	}

	return TypeDeclaration{}, fmt.Errorf("Non(Struct|Interface) of type %q not found", typeName)
}

// GetStructSpec attempts to retrieve the TypeSpec and StructType if the value
// matches this.
func GetStructSpec(val interface{}) (*ast.TypeSpec, *ast.StructType, error) {
	rval, ok := val.(*ast.TypeSpec)
	if !ok {
		return nil, nil, errors.New("Not ast.TypeSpec type")
	}

	rstruct, ok := rval.Type.(*ast.StructType)
	if !ok {
		return nil, nil, errors.New("Not ast.StructType type for *ast.TypeSpec.Type")
	}

	return rval, rstruct, nil
}

// MapOutFields defines a function to return a map of field name and value
// pair for the giving struct.
func MapOutFields(item StructDeclaration, rootName, tagName, fallback string) (string, error) {
	vals, err := MapOutFieldsToMap(item, rootName, tagName, fallback)
	if err != nil {
		return "", err
	}

	var bu bytes.Buffer

	if _, err := gen.Map("string", "interface{}", vals).WriteTo(&bu); err != nil {
		return "", err
	}

	return bu.String(), nil
}

// FieldByFieldName defines a function to return actual name of field with the given tag name.
func FieldByFieldName(item StructDeclaration, fieldName string) (FieldDeclaration, error) {
	if item.Declr == nil {
		fmt.Printf("Receiving StructDeclaration without PackageDeclaration: %#v\n", item)
		return FieldDeclaration{}, errors.New("StructDeclaration has no PackageDeclaration field")
	}
	fields := Fields(GetFields(item, item.Declr))

	for _, field := range fields {
		if field.FieldName != fieldName {
			continue
		}

		return field, nil
	}

	return FieldDeclaration{}, fmt.Errorf("Field name %q for Struct %q", fieldName, item.Object.Name.Name)
}

// FieldFor defines a function to return actual name of field with the given tag name.
func FieldFor(item StructDeclaration, tag string, tagFieldName string) (FieldDeclaration, error) {
	if item.Declr == nil {
		fmt.Printf("Receiving StructDeclaration without PackageDeclaration: %#v\n", item)
		return FieldDeclaration{}, errors.New("StructDeclaration has no PackageDeclaration field")
	}

	fields := Fields(GetFields(item, item.Declr))

	wTags := fields.TagFor(tag)

	// Collect key field names from embedded first
	for _, tag := range wTags {
		if tag.Value != tagFieldName {
			continue
		}

		return tag.Field, nil
	}

	return FieldDeclaration{}, fmt.Errorf("Tag value %q not found in tag %q for Struct %q", tagFieldName, tag, item.Object.Name.Name)
}

// FieldNameFor defines a function to return actual name of field with the given tag name.
func FieldNameFor(item StructDeclaration, tag string, tagFieldName string) string {
	if item.Declr == nil {
		fmt.Printf("Receiving StructDeclaration without PackageDeclaration: %#v\n", item)
		return ""
	}
	fields := Fields(GetFields(item, item.Declr))

	wTags := fields.TagFor(tag)

	// Collect key field names from embedded first
	for _, tag := range wTags {
		if tag.Value != tagFieldName {
			continue
		}

		return tag.Field.FieldName
	}

	return ""
}

// AssignDefaultValue will get the fieldName for a giving tag and tagVal and return	a string of giving
// variable name with fieldName equal to default value.
func AssignDefaultValue(item StructDeclaration, tag string, tagVal string, varName string) (string, error) {
	fieldName, defaultVal, err := DefaultFieldValueFor(item, tag, tagVal)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s = %s", varName, fieldName, defaultVal), nil
}

// DefaultFieldValueFor defines a function to return a field default value.
func DefaultFieldValueFor(item StructDeclaration, tag string, tagVal string) (string, string, error) {
	if item.Declr == nil {
		fmt.Printf("Receiving StructDeclaration without PackageDeclaration: %#v\n", item)
		return "", "", errors.New("StructDeclaration has no PackageDeclaration field")
	}
	fields := Fields(GetFields(item, item.Declr))

	wTags := fields.TagFor(tag)

	// Collect key field names from embedded first
	for _, tag := range wTags {
		if tag.Value != tagVal {
			continue
		}

		return tag.Field.FieldName, RandomFieldValue(tag.Field), nil
	}

	return "", "", fmt.Errorf("Field for tag value %q not found", tagVal)
}

// RandomFieldAssign generates a random Field of a giving struct and returns a variable assignment
// declaration with the types default value.
func RandomFieldAssign(item StructDeclaration, varName string, tag string, exceptions ...string) (string, error) {
	randomFieldVal, _, err := RandomFieldWithExcept(item, tag, exceptions...)
	if err != nil {
		return "", err
	}

	return AssignDefaultValue(item, tag, randomFieldVal, varName)
}

// RandomFieldWithExcept defines a function to return a random field name which is not
// included in the exceptions set.
func RandomFieldWithExcept(item StructDeclaration, tag string, exceptions ...string) (string, string, error) {
	if item.Declr == nil {
		fmt.Printf("Receiving StructDeclaration without PackageDeclaration: %#v\n", item)
		return "", "", errors.New("StructDeclaration has no PackageDeclaration field")
	}

	fields := Fields(GetFields(item, item.Declr))

	wTags := fields.TagFor(tag)

	// Collect key field names from embedded first
	{
	ml:
		for _, tag := range wTags {
			for _, exception := range exceptions {
				if tag.Value == exception {
					continue ml
				}
			}

			return tag.Value, tag.Field.FieldName, nil
		}

	}

	return "", "", errors.New("All tags match exceptions")
}

// MapOutFieldsToMap defines a function to return a map of field name and value
// pair for the giving struct.
func MapOutFieldsToMap(item StructDeclaration, rootName, tagName, fallback string) (map[string]io.WriterTo, error) {
	if item.Declr == nil {
		fmt.Printf("Receiving StructDeclaration without PackageDeclaration: %#v\n", item)
		return nil, errors.New("StructDeclaration has no PackageDeclaration field")
	}

	fields := Fields(GetFields(item, item.Declr))

	wTags := fields.TagFor(tagName)
	if len(wTags) == 0 {
		wTags = fields.TagFor(fallback)

		if len(wTags) == 0 {
			return nil, fmt.Errorf("No tags match for %q and %q fallback for struct %q", tagName, fallback, item.Object.Name)
		}
	}

	dm := make(map[string]io.WriterTo)

	embedded := fields.Embedded()

	for _, embed := range embedded {
		emt, ems, err := GetStructSpec(embed.Type.Decl)
		if err != nil {
			return nil, err
		}

		vals, err := MapOutFieldsToMap(StructDeclaration{
			Object: emt,
			Struct: ems,
			Declr:  item.Declr,
		}, fmt.Sprintf("%s.%s", rootName, embed.FieldName), tagName, fallback)

		if err != nil {
			return nil, err
		}

		for name, val := range vals {
			dm[name] = val
		}
	}

	// Collect key field names from embedded first
	for _, tag := range wTags {
		if tag.Value == "-" {
			continue
		}

		if tag.Field.Type != nil {
			embededType, embedStruct, err := GetStructSpec(tag.Field.Type.Decl)
			if err != nil {
				return nil, err
			}

			flds, err := MapOutFieldsToMap(StructDeclaration{
				Object: embededType,
				Struct: embedStruct,
				Declr:  item.Declr,
			}, fmt.Sprintf("%s.%s", rootName, tag.Field.FieldName), tagName, fallback)

			if err != nil {
				return nil, err
			}

			dm[tag.Value] = gen.Map("string", "interface{}", flds)
			continue
		}

		dm[tag.Value] = gen.Fmt("%s.%s", rootName, tag.Field.FieldName)
	}

	return dm, nil
}

// MapOutFieldsToJSON returns the giving map values containing string for the giving
// output.
func MapOutFieldsToJSON(item StructDeclaration, tagName, fallback string) (string, error) {
	document, err := MapOutFieldsToJSONWriter(item, tagName, fallback)
	if err != nil {
		return "", err
	}

	var doc bytes.Buffer

	if _, err := document.WriteTo(&doc); err != nil && err != io.EOF {
		return "", err
	}

	return doc.String(), nil
}

// MapOutFieldsWithRandomValuesToJSON returns the giving map values containing string for the giving
// output.
func MapOutFieldsWithRandomValuesToJSON(item StructDeclaration, tagName, fallback string) (string, error) {
	document, err := MapOutFieldsToJSONWriterWithRandomValues(item, tagName, fallback)
	if err != nil {
		return "", err
	}

	var doc bytes.Buffer

	if _, err := document.WriteTo(&doc); err != nil && err != io.EOF {
		return "", err
	}

	return doc.String(), nil
}

// MapOutFieldsToJSONWriter returns the giving map values containing string for the giving
// output.
func MapOutFieldsToJSONWriter(item StructDeclaration, tagName, fallback string) (io.WriterTo, error) {
	if item.Declr == nil {
		fmt.Printf("Receiving StructDeclaration without PackageDeclaration: %#v\n", item)
		return bytes.NewBuffer(nil), errors.New("StructDeclaration has no PackageDeclaration field")
	}

	fields := Fields(GetFields(item, item.Declr))

	wTags := fields.TagFor(tagName)
	if len(wTags) == 0 {
		wTags = fields.TagFor(fallback)

		if len(wTags) == 0 {
			return nil, fmt.Errorf("No tags match for %q and %q fallback for struct %q", tagName, fallback, item.Object.Name)
		}
	}

	documents := make(map[string]io.WriterTo)

	// Collect key field names from embedded first
	for _, tag := range wTags {
		if tag.Value == "-" {
			continue
		}

		if tag.Field.Type != nil {
			embededType, embedStruct, err := GetStructSpec(tag.Field.Type.Decl)
			if err != nil {
				return nil, err
			}

			document, err := MapOutFieldsToJSONWriter(StructDeclaration{
				Object: embededType,
				Struct: embedStruct,
				Declr:  item.Declr,
			}, tagName, fallback)

			if err != nil {
				return nil, err
			}

			documents[tag.Value] = document
			continue
		}

		valueJSON := DefaultTypeValueString(strings.ToLower(tag.Field.FieldTypeName))
		documents[tag.Value] = gen.Text(valueJSON)
	}

	return gen.JSONDocument(documents), nil
}

// MapOutFieldsToJSONWriterWithRandomValues returns the giving map values containing string for the giving
// output.
func MapOutFieldsToJSONWriterWithRandomValues(item StructDeclaration, tagName, fallback string) (io.WriterTo, error) {
	if item.Declr == nil {
		fmt.Printf("Receiving StructDeclaration without PackageDeclaration: %#v\n", item)
		return bytes.NewBuffer(nil), errors.New("StructDeclaration has no PackageDeclaration field")
	}

	fields := Fields(GetFields(item, item.Declr))

	wTags := fields.TagFor(tagName)
	if len(wTags) == 0 {
		wTags = fields.TagFor(fallback)

		if len(wTags) == 0 {
			return nil, fmt.Errorf("No tags match for %q and %q fallback for struct %q", tagName, fallback, item.Object.Name)
		}
	}

	documents := make(map[string]io.WriterTo)

	// Collect key field names from embedded first
	for _, tag := range wTags {
		if tag.Value == "-" {
			continue
		}

		if tag.Field.Type != nil {
			embededType, embedStruct, err := GetStructSpec(tag.Field.Type.Decl)
			if err != nil {
				return nil, err
			}

			document, err := MapOutFieldsToJSONWriter(StructDeclaration{
				Object: embededType,
				Struct: embedStruct,
				Declr:  item.Declr,
			}, tagName, fallback)

			if err != nil {
				return nil, err
			}

			documents[tag.Value] = document
			continue
		}

		valueJSON := RandomDataTypeValueJSON(tag.Field.FieldTypeName, tag.Field.FieldName)
		if valueJSON == "nil" {
			valueJSON = "null"
		}
		documents[tag.Value] = gen.Text(valueJSON)
	}

	return gen.JSONDocument(documents), nil
}

// MapOutValues defines a function to return a map of field name and associated
// placeholders as value.
func MapOutValues(item StructDeclaration, onlyExported bool) (string, error) {
	var bu bytes.Buffer

	if _, err := MapOutFieldsValues(item, onlyExported, nil).WriteTo(&bu); err != nil {
		return "", err
	}

	return bu.String(), nil
}

// MapOutFieldsValues defines a function to return a map of field name and associated
// placeholders as value.
func MapOutFieldsValues(item StructDeclaration, onlyExported bool, name *gen.NameDeclr) io.WriterTo {
	if item.Declr == nil {
		fmt.Printf("Receiving StructDeclaration without PackageDeclaration: %#v\n", item)
		return bytes.NewBuffer(nil)
	}

	fields := Fields(GetFields(item, item.Declr))

	var writers []io.WriterTo

	if name == nil {
		tmpName := gen.FmtName("%sVar", strings.ToLower(item.Object.Name.Name))

		name = &tmpName

		vardecl := gen.VarType(
			tmpName,
			gen.Type(item.Object.Name.Name),
		)

		writers = append(writers, vardecl, gen.Text("\n"))
	}

	normals := fields.Normal()
	embedded := fields.Embedded()

	handleOtherField := func(embed FieldDeclaration) {
		elemValue := gen.AssignValue(
			gen.FmtName("%s.%s", name.Name, embed.FieldName),
			gen.Text(DefaultTypeValueString(embed.FieldTypeName)),
		)

		writers = append(writers, elemValue, gen.Text("\n"))
	}

	handleStructField := func(embed FieldDeclaration) {
		embedName := gen.FmtName("%sVar", strings.ToLower(embed.FieldName))

		elemDeclr := gen.VarType(
			embedName,
			gen.Type(embed.FieldTypeName),
		)

		writers = append(writers, elemDeclr)

		if item.Struct != nil {
			body := MapOutFieldsValues(StructDeclaration{
				Object: embed.Spec,
				Struct: embed.Struct,
				Declr:  item.Declr,
			}, onlyExported, &embedName)

			writers = append(writers, body)
		}

		elemValue := gen.AssignValue(
			gen.FmtName("%s.%s", name.Name, embed.FieldName),
			embedName,
		)

		writers = append(writers, elemValue, gen.Text("\n"))
	}

	for _, embed := range embedded {
		if !embed.Exported && onlyExported {
			continue
		}

		if embed.IsStruct {
			handleStructField(embed)
			continue
		}

		handleOtherField(embed)
	}

	for _, normal := range normals {
		if !normal.Exported && onlyExported {
			continue
		}

		if normal.IsStruct {
			handleStructField(normal)
			continue
		}

		handleOtherField(normal)
	}

	return gen.Block(writers...)
}

// RandomFieldValue returns the default value for a giving field.
func RandomFieldValue(fld FieldDeclaration) string {
	return RandomDataTypeValueWithName(fld.FieldTypeName, fld.FieldName)
}

// DefaultFieldValue returns the default value for a giving field.
func DefaultFieldValue(fld FieldDeclaration) string {
	return DefaultTypeValueString(fld.FieldTypeName)
}

// RandomDataTypeValueJSON returns the default value string of a giving
// typeName.
func RandomDataTypeValueJSON(typeName string, varName string) string {
	switch typeName {
	case "time.Time", "*time.Time", "Time", "time.time":
		return strconv.Quote(time.Now().UTC().String())
	case "uint", "uint32", "uint64":
		return fmt.Sprintf("%d", rand.Uint64())
	case "bool":
		return fmt.Sprintf("%t", rand.Int63n(1) == 0)
	case "string":
		switch strings.ToLower(varName) {
		case "username", "user_name", "login_name":
			return fmt.Sprintf("%q", fake.UserName())
		case "user-agent", "useragent":
			return fmt.Sprintf("%q", fake.UserAgent())
		case "domain", "url":
			return fmt.Sprintf("%q", fake.DomainName())
		case "zip", "zip_code", "zip-code":
			return fmt.Sprintf("%q", fake.Zip())
		case "title", "user_title":
			return fmt.Sprintf("%q", fake.Title())
		case "day":
			return fmt.Sprintf("%q", fake.WeekDay())
		case "week":
			return fmt.Sprintf("%d Week", fake.WeekdayNum())
		case "year":
			return fmt.Sprintf("%d", fake.Year(1998, 10000))
		case "date", "date_time", "time":
			return strconv.Quote(time.Now().UTC().String())
		case "location", "location_address", "location_addr":
			return fmt.Sprintf("%q", fake.Street())
		case "company", "company_name", "companyname":
			return fmt.Sprintf("%q", fake.Company())
		case "subject", "subject_name", "subjectname":
			return fmt.Sprintf("%q", fake.EmailSubject())
		case "email", "email_address", "emailaddress":
			return fmt.Sprintf("%q", fake.EmailAddress())
		case "addr", "address", "streetAddress", "street_address", "main_address", "mainaddress", "streetaddress":
			return fmt.Sprintf("%q", fake.StreetAddress())
		case "companyaddress", "company_address":
			return fmt.Sprintf("%q", fake.StreetAddress())
		case "first_name", "firstname":
			return fmt.Sprintf("%q", fake.FirstName())
		case "last_name", "lastname":
			return fmt.Sprintf("%q", fake.LastName())
		case "name", "fullname", "full_name":
			return fmt.Sprintf("%q", fake.FullName())
		case "public_id", "publicid", "private_id", "privateid", "user_id", "tenant_user_id", "tenant_id", "user_tenant_id":
			return fmt.Sprintf("%q", fake.CharactersN(30))
		case "creditcardnum", "credit_card_number", "credit_card_num":
			return fmt.Sprintf("%q", fake.CreditCardNum(fake.CreditCardType()))
		case "creditcard", "credit_card":
			return fmt.Sprintf("%q", fake.CreditCardNum(fake.CreditCardType()))
		default:
			return fmt.Sprintf("%q", fake.CharactersN(20))
		}
	case "rune":
		return fmt.Sprintf("'%x'", fake.CharactersN(1))
	case "byte":
		return fmt.Sprintf("'%x'", fake.CharactersN(1))
	case "float32", "float64":
		return fmt.Sprintf("%.4f", rand.Float64())
	case "int", "int32", "int64":
		switch varName {
		case "week":
			return fmt.Sprintf("%d", fake.WeekdayNum())
		case "day":
			return fmt.Sprintf("%d", fake.Day())
		case "year":
			return fmt.Sprintf("%d", fake.Year(1998, 10000))
		default:
			return fmt.Sprintf("%d", rand.Int63n(20))
		}
	default:
		return "null"
	}
}

// RandomDataTypeValueWithName returns the default value string of a giving
// typeName.
func RandomDataTypeValueWithName(typeName string, varName string) string {
	switch typeName {
	case "time.Time", "*time.Time", "Time", "time.time":
		return strconv.Quote(time.Now().UTC().String())
	case "uint", "uint32", "uint64":
		return fmt.Sprintf("%d", rand.Uint64())
	case "bool":
		return fmt.Sprintf("%t", rand.Int63n(1) == 0)
	case "string":
		switch strings.ToLower(varName) {
		case "username", "user_name", "login_name":
			return fmt.Sprintf("%q", fake.UserName())
		case "user-agent", "useragent":
			return fmt.Sprintf("%q", fake.UserAgent())
		case "domain", "url":
			return fmt.Sprintf("%q", fake.DomainName())
		case "zip", "zip_code", "zip-code":
			return fmt.Sprintf("%q", fake.Zip())
		case "title", "user_title":
			return fmt.Sprintf("%q", fake.Title())
		case "day":
			return fmt.Sprintf("%q", fake.WeekDay())
		case "week":
			return fmt.Sprintf("%d Week", fake.WeekdayNum())
		case "year":
			return fmt.Sprintf("%d", fake.Year(1998, 10000))
		case "date", "date_time", "time":
			return strconv.Quote(time.Now().UTC().String())
		case "location", "location_address", "location_addr":
			return fmt.Sprintf("%q", fake.Street())
		case "company", "company_name", "companyname":
			return fmt.Sprintf("%q", fake.Company())
		case "subject", "subject_name", "subjectname":
			return fmt.Sprintf("%q", fake.EmailSubject())
		case "email", "email_address", "emailaddress":
			return fmt.Sprintf("%q", fake.EmailAddress())
		case "addr", "address", "streetAddress", "street_address", "main_address", "mainaddress", "streetaddress":
			return fmt.Sprintf("%q", fake.StreetAddress())
		case "companyaddress", "company_address":
			return fmt.Sprintf("%q", fake.StreetAddress())
		case "first_name", "firstname":
			return fmt.Sprintf("%q", fake.FirstName())
		case "last_name", "lastname":
			return fmt.Sprintf("%q", fake.LastName())
		case "name", "fullname", "full_name":
			return fmt.Sprintf("%q", fake.FullName())
		case "public_id", "publicid", "private_id", "privateid", "user_id", "tenant_user_id", "tenant_id", "user_tenant_id":
			return fmt.Sprintf("%q", fake.CharactersN(30))
		case "creditcardnum", "credit_card_number", "credit_card_num":
			return fmt.Sprintf("%q", fake.CreditCardNum(fake.CreditCardType()))
		case "creditcard", "credit_card":
			return fmt.Sprintf("%q", fake.CreditCardNum(fake.CreditCardType()))
		default:
			return fmt.Sprintf("%q", fake.CharactersN(20))
		}
	case "rune":
		return fmt.Sprintf("'%x'", fake.CharactersN(1))
	case "byte":
		return fmt.Sprintf("'%x'", fake.CharactersN(1))
	case "float32", "float64":
		return fmt.Sprintf("%.4f", rand.Float64())
	case "int", "int32", "int64":
		switch varName {
		case "week":
			return fmt.Sprintf("%d", fake.WeekdayNum())
		case "day":
			return fmt.Sprintf("%d", fake.Day())
		case "year":
			return fmt.Sprintf("%d", fake.Year(1998, 10000))
		default:
			return fmt.Sprintf("%d", rand.Int63n(20))
		}
	default:
		return DefaultTypeValueString(typeName)
	}
}

// RandomDataTypeValue returns the default value string of a giving
// typeName.
func RandomDataTypeValue(typeName string) string {
	switch typeName {
	case "time.Time":
		return time.Now().UTC().String()
	case "uint", "uint32", "uint64":
		return fmt.Sprintf("%d", rand.Uint64())
	case "bool":
		return fmt.Sprintf("%t", rand.Int63n(1) == 0)
	case "string":
		return fmt.Sprintf("%q", fake.Character())
	case "rune":
		return fmt.Sprintf("'%x'", fake.CharactersN(1))
	case "byte":
		return fmt.Sprintf("'%x'", fake.CharactersN(1))
	case "float32", "float64":
		return fmt.Sprintf("%.4f", rand.Float64())
	case "int", "int32", "int64":
		return fmt.Sprintf("%d", rand.Int63())
	default:
		return DefaultTypeValueString(typeName)
	}
}

// DefaultTypeValueString returns the default value string of a giving
// typeName.
func DefaultTypeValueString(typeName string) string {
	switch typeName {
	case "uint", "uint32", "uint64":
		return "0"
	case "bool":
		return `false`
	case "time.Time", "*time.Time", "Time", "time.time":
		return strconv.Quote(time.Now().UTC().String())
	case "string":
		return `""`
	case "rune":
		return `rune(0)`
	case "[]uint":
		return `[]uint{}`
	case "[]uint64":
		return `[]uint64{}`
	case "[]uint32":
		return `[]uint32{}`
	case "[]int":
		return `[]int{}`
	case "[]int64":
		return `[]int64{}`
	case "[]int32":
		return `[]int32{}`
	case "[]bool":
		return `[]bool{}`
	case "[]string":
		return `[]string{}`
	case "[]byte":
		return `[]byte{}`
	case "byte":
		return `byte(rune(0))`
	case "float32", "float64":
		return "0.0"
	case "int", "int32", "int64":
		return "0"
	case "map[string]interface{}":
		return "map[string]interface{}"
	case "map[string]string":
		return "map[string]string{}"
	default:
		return "nil"
	}
}

// GetTag returns the giving tag associated with the name if it exists.
func GetTag(f FieldDeclaration, tagName string, fallback string) (TagDeclaration, error) {
	tg, err := f.GetTag(tagName)
	if err != nil {
		return f.GetTag(fallback)
	}

	return tg, nil
}
