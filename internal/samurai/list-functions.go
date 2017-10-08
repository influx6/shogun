package samurai

import (
	"fmt"
	"go/doc"
	"os"

	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/vfiles"
	"github.com/influx6/gobuild/build"
	"github.com/influx6/moz/ast"
	"github.com/influx6/shogun/internal"
)

// ListFunctions returns all functions retrieved from the directory filtered by the build.Context.
func ListFunctions(vlog, events metrics.Metrics, targetDir string, ctx build.Context) ([]internal.Function, error) {
	// Build shogunate directory itself first.
	functions, err := ListFunctionsForDir(vlog, events, targetDir, ctx)
	if err != nil {
		events.Emit(metrics.Errorf("Failed to generate function list : %+q", err))
		return nil, err
	}

	if err := vfiles.WalkDirSurface(targetDir, func(rel string, abs string, info os.FileInfo) error {
		if !info.IsDir() {
			return nil
		}

		res, err := ListFunctionsForDir(vlog, events, abs, ctx)
		if err != nil {
			return err
		}

		functions = append(functions, res...)
		return nil
	}); err != nil {
		events.Emit(metrics.Error(err).With("dir", targetDir))
		return functions, err
	}

	return functions, nil
}

// ListFunctionsForDir iterates all directories and retrieves functon list of all declared functions
// matching the shegun format.
func ListFunctionsForDir(vlog, events metrics.Metrics, dir string, ctx build.Context) ([]internal.Function, error) {
	pkgs, err := ast.FilteredPackageWithBuildCtx(vlog, dir, ctx)
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", dir))
		if _, ok := err.(*build.NoGoError); ok {
			return nil, nil
		}
		return nil, err
	}

	var functions []internal.Function

	for _, pkgItem := range pkgs {
		for _, declr := range pkgItem.Packages {
			for _, function := range declr.Functions {
				def, err := function.Definition(&declr)
				if err != nil {
					return nil, err
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
					continue
				}

				// If the Context is unknown then skip.
				if contextType == internal.UseUnknownContext {
					continue
				}

				// If the return format is unknown then skip.
				if returnType == internal.UnknownErrorReturn {
					continue
				}

				var fn internal.Function
				fn.Name = def.Name
				fn.Description = function.Comments
				fn.Synopses = doc.Synopsis(function.Comments)

				fmt.Printf("Name: %q - Synopse: %q - Imports: %+q\n", fn.Name, fn.Synopses, importList)

				functions = append(functions, fn)
			}
		}
	}

	return functions, nil
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
		if arg.StructObject != nil {
			if arg2 == nil {
				return internal.WithStructArgument, []internal.VarMeta{
					{
						Type:       arg.Type,
						TypeAddr:   arg.ExType,
						Import:     arg.Import.Path,
						ImportNick: arg.Import.Name,
					},
				}
			}

			if arg2.Type == ioWriteCloser {
				return internal.WithStructAndWriteCloserArgument, nil
			}
		}

		if arg.InterfaceObject != nil {
			if arg2 == nil {
				return internal.WithInterfaceArgument, []internal.VarMeta{
					{
						Type:       arg.Type,
						TypeAddr:   arg.ExType,
						Import:     arg.Import.Path,
						ImportNick: arg.Import.Name,
					},
				}
			}

			if arg2.Type == ioWriteCloser {
				return internal.WithInterfaceAndWriteCloserArgument, nil
			}
		}

		if arg.ImportedObject != nil {
			if arg2 == nil {
				return internal.WithImportedAndWriteCloserArgument, []internal.VarMeta{
					{
						Type:       arg.Type,
						TypeAddr:   arg.ExType,
						Import:     arg.Import.Path,
						ImportNick: arg.Import.Name,
					},
				}
			}

			if arg2.Type == ioWriteCloser {
				return internal.WithInterfaceAndWriteCloserArgument, nil
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
