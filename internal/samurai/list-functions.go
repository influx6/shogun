package samurai

import (
	"go/doc"
	"os"

	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/vfiles"
	"github.com/influx6/gobuild/build"
	"github.com/influx6/moz/ast"
	"github.com/influx6/shogun/internal"
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
			return err2
		}

		list.Subs[rel] = res
		return nil
	}); err != nil {
		events.Emit(metrics.Error(err).With("dir", targetDir))
		return list, err
	}

	return list, nil
}

// PackageFunctionList holds the list of processed functions from individual packages.
type PackageFunctionList struct {
	Path string
	Hash string
	List []internal.Function
}

// ListFunctionsForDir iterates all directories and retrieves functon list of all declared functions
// matching the shegun format.
func ListFunctionsForDir(vlog, events metrics.Metrics, dir string, ctx build.Context) (PackageFunctionList, error) {
	var pkgFuncs PackageFunctionList
	pkgFuncs.Path = dir

	pkgs, err := ast.FilteredPackageWithBuildCtx(vlog, dir, ctx)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			return pkgFuncs, nil
		}

		events.Emit(metrics.Error(err).With("dir", dir))
		return pkgFuncs, err
	}

	var hash []byte

	for _, pkgItem := range pkgs {
		pkgHash, err := generateHash(pkgItem.Files)
		if err != nil {
			return pkgFuncs, err
		}

		hash = append(hash, []byte(pkgHash)...)

		fns, err := pullFunctions(pkgItem)
		if err != nil {
			return pkgFuncs, err
		}

		pkgFuncs.List = append(pkgFuncs.List, fns...)
	}

	pkgFuncs.Hash = string(hash)

	return pkgFuncs, nil
}

func pullFunctions(pkg ast.Package) ([]internal.Function, error) {
	var list []internal.Function

	for _, declr := range pkg.Packages {
		for _, function := range declr.Functions {
			fn, ignore, err := pullFunction(&function, &declr)
			if err != nil {
				return list, err
			}

			if ignore {
				continue
			}

			list = append(list, fn)
		}
	}

	return list, nil
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

	var importList []internal.VarMeta

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
		contextType = getContextState(def.Args[0])
		if contextType == internal.UseUnknownContext {
			contextType = internal.NoContext
			argumentType, importList = getArgumentsState(def.Args[0], nil)
		}
	case 2:
		contextType = getContextState(def.Args[0])
		if contextType == internal.UseUnknownContext {
			contextType = internal.NoContext
			argumentType, importList = getArgumentsState(def.Args[0], &def.Args[1])
		} else {
			argumentType, importList = getArgumentsState(def.Args[1], nil)
		}
	case 3:
		contextType = getContextState(def.Args[0])
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

	fn.Name = def.Name
	fn.Imports = importList
	fn.Type = argumentType
	fn.Return = returnType
	fn.Context = contextType
	fn.Package = function.Package
	fn.PackagePath = function.Path
	fn.PackageFile = function.FilePath
	fn.PackageFileName = function.File
	fn.Description = function.Comments
	fn.Synopses = doc.Synopsis(function.Comments)

	if depends, ok := function.GetAnnotation("@depends"); ok {
		fn.Depends = append(fn.Depends, depends.Arguments...)
	}

	return fn, false, nil
}

var ioWriteCloser = "io.WriteCloser"

func getArgumentsState(arg ast.ArgType, arg2 *ast.ArgType) (int, []internal.VarMeta) {
	switch arg.Type {
	case "io.Reader":
		if arg2 == nil {
			return internal.WithReaderArgument, nil
		}

		if arg2.Type == ioWriteCloser {
			return internal.WithReaderAndWriteCloserArgument, nil
		}

		return internal.WithUnknownArgument, nil
	case "io.WriteCloser":
		if arg2 == nil {
			return internal.WithWriteCloserArgument, nil
		}
		return internal.WithUnknownArgument, nil
	case "map[string]interface{}":
		if arg2 == nil {
			return internal.WithMapArgument, nil
		}
		return internal.WithUnknownArgument, nil
	default:
		params := []internal.VarMeta{
			{
				Type:       arg.Type,
				TypeAddr:   arg.ExType,
				Import:     arg.Import.Path,
				ImportNick: arg.Import.Name,
			},
		}

		if arg.StructObject != nil {
			if arg2 == nil {
				return internal.WithStructArgument, params
			}

			if arg2.Type == ioWriteCloser {
				return internal.WithStructAndWriteCloserArgument, params
			}
		}

		if arg.InterfaceObject != nil {
			if arg2 == nil {
				return internal.WithInterfaceArgument, params
			}

			if arg2.Type == ioWriteCloser {
				return internal.WithInterfaceAndWriteCloserArgument, params
			}
		}

		if arg.ImportedObject != nil {
			if arg2 == nil {
				return internal.WithImportedAndWriteCloserArgument, params
			}

			if arg2.Type == ioWriteCloser {
				return internal.WithInterfaceAndWriteCloserArgument, params
			}
		}
	}

	return internal.WithUnknownArgument, nil
}

func getReturnState(arg ast.ArgType) int {
	switch arg.Type {
	case "error":
		return internal.ErrorReturn
	}

	return internal.UnknownErrorReturn
}

func getContextState(arg ast.ArgType) int {
	switch arg.Type {
	case "context.Context":
		if arg.Import.Path == "context" {
			return internal.UseGoogleContext
		}
		return internal.UseUnknownContext
	case "context.CancelContext":
		if arg.Import.Path == "github.com/influx6/faux/context" {
			return internal.UseFauxCancelContext
		}
		return internal.UseUnknownContext
	case "context.ValueBagContext":
		if arg.Import.Path == "github.com/influx6/faux/context" {
			return internal.UseValueBagContext
		}
		return internal.UseUnknownContext
	}

	return internal.UseUnknownContext
}
