package samurai

import (
	"bytes"
	"errors"
	"fmt"
	"go/doc"
	"os"
	"strings"
	"text/template"

	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/vfiles"
	"github.com/influx6/gobuild/build"
	"github.com/influx6/gobuild/srcpath"
	"github.com/influx6/moz/ast"
	"github.com/influx6/moz/gen"
	"github.com/influx6/shogun/internal"
	"github.com/influx6/shogun/templates"
)

const (
	defaultDesc   = "No description provided."
	fauxContext   = "github.com/influx6/faux/context"
	googleContext = "context"
)

// errors.
var (
	ErrSkipDir = errors.New("Directory does not match build type, skip")
)

// FunctionList holds a list of functions from a main package and
// all other subpackages retrieved.
type FunctionList struct {
	Dir  string
	Main PackageFunctionList
	Subs map[string]PackageFunctionList
}

// ListFunctions returns all functions retrieved from the directory filtered by the build.Context.
func ListFunctions(vlog, events metrics.Metrics, targetDir string, ctx build.Context) (FunctionList, error) {
	var list FunctionList
	list.Dir = targetDir
	list.Subs = make(map[string]PackageFunctionList)

	// Build shogunate directory itself first.
	var err error
	list.Main, err = ListFunctionsForDir(vlog, events, targetDir, ctx)
	if err != nil {
		events.Emit(metrics.Errorf("Failed to generate function list : %+q", err))
		return list, err
	}

	if err = vfiles.WalkDirSurface(targetDir, func(rel string, abs string, info os.FileInfo) error {
		if !info.IsDir() {
			return nil
		}

		res, err2 := ListFunctionsForDir(vlog, events, abs, ctx)
		if err2 != nil {
			if err2 == ErrSkipDir {
				return nil
			}

			return err2
		}

		res.RelPath = rel
		list.Subs[res.Path] = res
		return nil
	}); err != nil {
		events.Emit(metrics.Error(err).With("dir", targetDir))
		return list, err
	}

	return list, nil
}

// PackageFunctionList holds the list of processed functions from individual packages.
type PackageFunctionList struct {
	Path    string
	RelPath string
	Hash    string
	Package string
	Name    string
	Desc    string
	List    []internal.PackageFunctions
}

// ListFunctionsForDir iterates all directories and retrieves functon list of all declared functions
// matching the shegun format.
func ListFunctionsForDir(vlog, events metrics.Metrics, dir string, ctx build.Context) (PackageFunctionList, error) {
	var pkgFuncs PackageFunctionList
	pkgFuncs.Path = dir
	pkgFuncs.Package, _ = srcpath.RelativeToSrc(dir)

	pkgs, err := ast.FilteredPackageWithBuildCtx(vlog, dir, ctx)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			return pkgFuncs, ErrSkipDir
		}

		events.Emit(metrics.Error(err).With("dir", dir))
		return pkgFuncs, err
	}

	if len(pkgs) == 0 {
		return pkgFuncs, ErrSkipDir
	}

	pkgItem := pkgs[0]
	if pkgItem.HasAnnotation("@shogunIgnore") {
		return pkgFuncs, ErrSkipDir
	}

	pkgHash, err := generateHash(pkgItem.Files)
	if err != nil {
		return pkgFuncs, err
	}

	binaryName := pkgItem.Name

	var binaryDesc string
	if binAnnon, _, ok := pkgItem.AnnotationFirstFor("@binaryName"); ok {
		binaryName = binAnnon.Param("name")
		if binaryName == "" {
			binaryName = pkgItem.Name
		}

		desc := binAnnon.Param("desc")
		if desc != "" && !strings.HasSuffix(desc, ".") {
			desc += "."
			binaryDesc = doc.Synopsis(desc)
		}
	}

	if binaryDesc == "" {
		binaryDesc = haiku()
	}

	pkgFuncs.Name = binaryName
	pkgFuncs.Desc = binaryDesc

	fns, err := pullFunctions(pkgItem)
	if err != nil {
		return pkgFuncs, err
	}

	fns.Hash = pkgHash
	pkgFuncs.List = append(pkgFuncs.List, fns)

	return pkgFuncs, nil
}

func pullFunctions(pkg ast.Package) (internal.PackageFunctions, error) {
	var fnPkg internal.PackageFunctions
	fnPkg.Name = pkg.Name
	fnPkg.Path = pkg.Path
	fnPkg.FilePath = pkg.FilePath

	if annon, _, found := pkg.AnnotationFirstFor("@binaryName"); found {
		desc := annon.Param("desc")
		if desc == "" {
			desc = haiku()
		}

		if desc != "" && !strings.HasSuffix(desc, ".") {
			desc += "."
		}

		fnPkg.Desc = doc.Synopsis(desc)
		fnPkg.BinaryName = annon.Param("name")
	}

	if fnPkg.BinaryName == "" {
		fnPkg.BinaryName = pkg.Name
	}

	for _, declr := range pkg.Packages {
		if declr.HasAnnotation("@shogunIgnoreFunctions") {
			continue
		}

		for _, function := range declr.Functions {
			fn, ignore, err := pullFunction(&function, &declr)
			if err != nil {
				return fnPkg, err
			}

			if ignore {
				continue
			}

			fnPkg.List = append(fnPkg.List, fn)
		}
	}

	fnPkg.MaxNameLen = maxName(fnPkg)

	return fnPkg, nil
}

// pullFunctionFromDeclr returns all function details within the giving PackageDeclaration.
func pullFunctionFromDeclr(pkg ast.Package, declr *ast.PackageDeclaration) ([]internal.Function, error) {
	var list []internal.Function

	for _, function := range declr.Functions {
		fn, ignore, err := pullFunction(&function, declr)
		if err != nil {
			return list, err
		}

		if ignore {
			continue
		}

		list = append(list, fn)
	}

	return list, nil
}

func pullFunction(function *ast.FuncDeclaration, declr *ast.PackageDeclaration) (internal.Function, bool, error) {
	var fn internal.Function

	if function.HasAnnotation("@ignore") {
		return fn, true, nil
	}

	def, err := function.Definition(declr)
	if err != nil {
		return fn, true, err
	}

	argLen := len(def.Args)
	retLen := len(def.Returns)

	var returnType int
	var argumentType int
	var contextType int

	var importList, ctxImport internal.VarMeta

	switch retLen {
	case 0:
		returnType = internal.NoReturn
	case 1:
		returnType = getReturnState(def.Returns[0])
	}

	switch argLen {
	case 0:
		contextType = internal.NoContext
		argumentType = internal.NoArgument
	case 1:
		contextType, ctxImport = getContextState(def.Args[0])
		if contextType == internal.UseUnknownContext {
			contextType = internal.NoContext
			argumentType, importList = getArgumentsState(def.Args[0], nil)
		} else {
			argumentType = internal.WithContextArgument
		}
	case 2:
		contextType, ctxImport = getContextState(def.Args[0])
		if contextType == internal.UseUnknownContext {
			contextType = internal.NoContext
			argumentType, importList = getArgumentsState(def.Args[0], &def.Args[1])
		} else {
			argumentType, importList = getArgumentsState(def.Args[1], nil)
		}
	case 3:
		contextType, ctxImport = getContextState(def.Args[0])
		argumentType, importList = getArgumentsState(def.Args[1], &def.Args[2])
	}

	// If the argument format does not match allowed, skip.
	if argumentType == internal.WithUnknownArgument {
		return fn, true, nil
	}

	// If the Context is unknown then skip.
	if contextType == internal.UseUnknownContext {
		return fn, true, nil
	}

	// If the return format is unknown then skip.
	if returnType == internal.UnknownErrorReturn {
		return fn, true, nil
	}

	fn.RealName = def.Name
	fn.Type = argumentType
	fn.Return = returnType
	fn.Imports = importList
	fn.Context = contextType
	fn.ContextImport = ctxImport
	fn.Source = function.Source
	fn.Package = function.Package
	fn.PackagePath = function.Path
	fn.Name = strings.ToLower(def.Name)
	fn.PackageFile = function.FilePath
	fn.PackageFileName = function.File
	fn.Description = function.Comments
	fn.Synopses = doc.Synopsis(function.Comments)

	if fn.Description == "" {
		fn.Description = defaultDesc
	}

	if fn.Synopses == "" {
		fn.Synopses = defaultDesc
	}

	if function.HasAnnotation("@default") {
		fn.Default = true
	}

	if depends, ok := function.GetAnnotation("@depends"); ok {
		fn.Depends = append(fn.Depends, depends.Arguments...)
	}

	var helpMessage bytes.Buffer
	if _, err := gen.SourceTextWithName(
		"shogun-pkg-fn-message",
		string(templates.Must("shogun-pkg-fn-message.tml")),
		template.FuncMap{
			"returnsError": func(d int) bool {
				return d == internal.ErrorReturn
			},
			"usesNoContext": func(d int) bool {
				return d == internal.NoContext
			},
			"usesGoogleContext": func(d int) bool {
				return d == internal.UseGoogleContext
			},
			"usesFauxContext": func(d int) bool {
				return d == internal.UseFauxCancelContext
			},
			"hasNoArgument": func(d int) bool {
				return d == internal.NoArgument
			},
			"hasContextArgument": func(d int) bool {
				return d == internal.WithContextArgument
			},
			"hasMapArgument": func(d int) bool {
				return d == internal.WithMapArgument
			},
			"hasStructArgument": func(d int) bool {
				return d == internal.WithStructArgument
			},
			"hasReadArgument": func(d int) bool {
				return d == internal.WithReaderArgument
			},
			"hasWriteArgument": func(d int) bool {
				return d == internal.WithWriteCloserArgument
			},
			"hasImportedArgument": func(d int) bool {
				return d == internal.WithImportedObjectArgument
			},
			"hasReadArgumentWithWriter": func(d int) bool {
				return d == internal.WithReaderAndWriteCloserArgument
			},
			"hasStructArgumentWithWriter": func(d int) bool {
				return d == internal.WithStructAndWriteCloserArgument
			},
			"hasMapArgumentWithWriter": func(d int) bool {
				return d == internal.WithMapAndWriteCloserArgument
			},
			"hasImportedArgumentWithWriter": func(d int) bool {
				return d == internal.WithImportedAndWriteCloserArgument
			},
		},
		fn,
	).WriteTo(&helpMessage); err != nil {
		return fn, false, fmt.Errorf("Failed to generate function's %q help message: %+q", fn.RealName, err)
	}

	var helpMessageWithSource bytes.Buffer
	if _, err := gen.SourceTextWithName(
		"shogun-pkg-fn-message-withsource",
		string(templates.Must("shogun-pkg-fn-message-withsource.tml")),
		template.FuncMap{
			"returnsError": func(d int) bool {
				return d == internal.ErrorReturn
			},
			"usesNoContext": func(d int) bool {
				return d == internal.NoContext
			},
			"usesGoogleContext": func(d int) bool {
				return d == internal.UseGoogleContext
			},
			"usesFauxContext": func(d int) bool {
				return d == internal.UseFauxCancelContext
			},
			"hasNoArgument": func(d int) bool {
				return d == internal.NoArgument
			},
			"hasContextArgument": func(d int) bool {
				return d == internal.WithContextArgument
			},
			"hasMapArgument": func(d int) bool {
				return d == internal.WithMapArgument
			},
			"hasStructArgument": func(d int) bool {
				return d == internal.WithStructArgument
			},
			"hasReadArgument": func(d int) bool {
				return d == internal.WithReaderArgument
			},
			"hasWriteArgument": func(d int) bool {
				return d == internal.WithWriteCloserArgument
			},
			"hasImportedArgument": func(d int) bool {
				return d == internal.WithImportedObjectArgument
			},
			"hasReadArgumentWithWriter": func(d int) bool {
				return d == internal.WithReaderAndWriteCloserArgument
			},
			"hasStructArgumentWithWriter": func(d int) bool {
				return d == internal.WithStructAndWriteCloserArgument
			},
			"hasMapArgumentWithWriter": func(d int) bool {
				return d == internal.WithMapAndWriteCloserArgument
			},
			"hasImportedArgumentWithWriter": func(d int) bool {
				return d == internal.WithImportedAndWriteCloserArgument
			},
		},
		fn,
	).WriteTo(&helpMessageWithSource); err != nil {
		return fn, false, fmt.Errorf("Failed to generate function's %q help message with source: %+q", fn.RealName, err)
	}

	fn.HelpMessage = helpMessage.String()
	fn.HelpMessageWithSource = helpMessageWithSource.String()

	return fn, false, nil
}

var ioWriteCloser = "io.WriteCloser"

func getArgumentsState(arg ast.ArgType, arg2 *ast.ArgType) (int, internal.VarMeta) {
	switch arg.Type {
	case "io.Reader":
		if arg2 == nil {
			return internal.WithReaderArgument, internal.VarMeta{}
		}

		if arg2.Type == ioWriteCloser {
			return internal.WithReaderAndWriteCloserArgument, internal.VarMeta{}
		}

		return internal.WithUnknownArgument, internal.VarMeta{}
	case "io.WriteCloser":
		if arg2 == nil {
			return internal.WithWriteCloserArgument, internal.VarMeta{}
		}
		return internal.WithUnknownArgument, internal.VarMeta{}
	case "map[string]interface{}":
		if arg2 == nil {
			return internal.WithMapArgument, internal.VarMeta{}
		}

		if arg2.Type == ioWriteCloser {
			return internal.WithMapAndWriteCloserArgument, internal.VarMeta{}
		}

		return internal.WithUnknownArgument, internal.VarMeta{}
	default:
		params := internal.VarMeta{
			Type:       arg.Type,
			TypeAddr:   arg.ExType,
			Import:     arg.Import.Path,
			ImportNick: arg.Import.Name,
		}

		if arg.ImportedObject != nil {
			if arg.StructObject == nil {
				return internal.WithUnknownArgument, internal.VarMeta{}
			}

			if arg2 == nil {
				return internal.WithImportedObjectArgument, params
			}

			if arg2.Type == ioWriteCloser {
				return internal.WithImportedAndWriteCloserArgument, params
			}
		}

		if arg.StructObject != nil {
			if arg2 == nil {
				return internal.WithStructArgument, params
			}

			if arg2.Type == ioWriteCloser {
				return internal.WithStructAndWriteCloserArgument, params
			}
		}
	}

	return internal.WithUnknownArgument, internal.VarMeta{}
}

func getReturnState(arg ast.ArgType) int {
	switch arg.Type {
	case "error":
		return internal.ErrorReturn
	}

	return internal.UnknownErrorReturn
}

func getContextState(arg ast.ArgType) (int, internal.VarMeta) {
	var imp internal.VarMeta
	imp.Type = arg.Type
	imp.TypeAddr = arg.ExType

	switch arg.Type {
	case "context.Context":
		if arg.Import.Path == googleContext {
			imp.Import = googleContext
			imp.ImportNick = arg.Import.Name
			return internal.UseGoogleContext, imp
		}
		return internal.UseUnknownContext, imp
	case "context.CancelContext":
		if arg.Import.Path == fauxContext {
			imp.Import = fauxContext
			imp.ImportNick = arg.Import.Name
			return internal.UseFauxCancelContext, imp
		}
		return internal.UseUnknownContext, imp
	}

	return internal.UseUnknownContext, imp
}

// maxName returns name of longest function.
func maxName(pn internal.PackageFunctions) int {
	curr := -1
	for _, fn := range pn.List {
		if curr < len(fn.Name) {
			curr = len(fn.Name)
			continue
		}
	}

	return curr
}
