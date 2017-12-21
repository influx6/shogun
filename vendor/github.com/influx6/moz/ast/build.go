package ast

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/types/actions"
	"github.com/influx6/faux/types/events"
	"github.com/influx6/gobuild/build"
	"github.com/influx6/gobuild/srcpath"
	"github.com/influx6/moz/gen"
)

var (
	// ErrEmptyList defines a error returned for a empty array or slice.
	ErrEmptyList = errors.New("Slice/List is empty")

	// ErrPackageParseFailed defines a error returned when a package processing failed to work.
	ErrPackageParseFailed = errors.New("Package or Package file failed to be parsed")

	// we need to ensure we catch all processed packages to ensure we dont get stuck
	// re-processing in a loop again.
	processedPackages = struct {
		pl   sync.Mutex
		pkgs map[string]Package
	}{
		pkgs: make(map[string]Package),
	}
)

// ParseFileAnnotations parses the package from the provided file.
func ParseFileAnnotations(log metrics.Metrics, path string) (Package, error) {
	return PackageFileWithBuildCtx(log, path, build.Default)
}

// ParseAnnotations parses the package which generates a series of ast with associated
// annotation for processing.
func ParseAnnotations(log metrics.Metrics, dir string) (Packages, error) {
	return PackageWithBuildCtx(log, dir, build.Default)
}

//===========================================================================================================

// FilteredPackageWithBuildCtx parses the package directory which generates a series of ast with associated
// annotation for processing by using the golang token parser, it uses the build.Context to
// collected context details for the package and only processes the files found by the build context.
// If you need something more broad without filtering, use PackageWithBuildCtx.
func FilteredPackageWithBuildCtx(log metrics.Metrics, dir string, ctx build.Context) (Packages, error) {
	rootbuildPkg, err := ctx.ImportDir(dir, 0)
	if err != nil {
		log.Emit(metrics.Errorf("Failed to retrieve build.Package for root directory"),
			metrics.With("file", dir),
			metrics.With("dir", dir),
			metrics.With("error", err.Error()),
			metrics.With("mode", build.FindOnly))
		return nil, err
	}

	if len(rootbuildPkg.GoFiles) == 0 {
		return nil, &build.NoGoError{}
	}

	log.Emit(metrics.Info("Generated build.Package"),
		metrics.With("file", dir),
		metrics.With("dir", dir),
		metrics.With("pkg", rootbuildPkg),
		metrics.With("mode", build.FindOnly))

	allowed := make(map[string]bool)
	for _, file := range rootbuildPkg.GoFiles {
		allowed[file] = true
	}

	filter := func(f os.FileInfo) bool {
		log.Emit(metrics.Info("Parse Filtering file"), metrics.With("incoming-file", f.Name()), metrics.With("allowed", allowed[f.Name()]))
		return allowed[f.Name()]
	}

	tokenFiles := token.NewFileSet()
	packages, err := parser.ParseDir(tokenFiles, dir, filter, parser.ParseComments)
	if err != nil {
		log.Emit(metrics.Error(err), metrics.With("message", "Failed to parse dir"), metrics.With("dir", dir))
		return nil, err
	}

	packageDeclrs := make(map[string]Package)
	packageBuilds := make(map[string]*build.Package)

	for tag, pkg := range packages {
		var pkgFiles []string

		for path, file := range pkg.Files {
			pkgFiles = append(pkgFiles, path)
			pathPkg := filepath.Dir(path)
			buildPkg, ok := packageBuilds[pathPkg]
			if !ok {
				buildPkg, err = ctx.ImportDir(pathPkg, 0)
				if err != nil {
					log.Emit(metrics.Errorf("Failed to retrieve build.Package"),
						metrics.With("file", path),
						metrics.With("dir", dir),
						metrics.With("file-dir", filepath.Dir(path)),
						metrics.With("error", err.Error()),
						metrics.With("mode", build.FindOnly))
				} else {
					packageBuilds[pathPkg] = buildPkg

					log.Emit(metrics.Info("Generated build.Package"),
						metrics.With("file", path),
						metrics.With("pkg", buildPkg),
						metrics.With("file-dir", filepath.Dir(path)),
						metrics.With("dir", dir),
						metrics.With("mode", build.FindOnly))
				}
			}

			res, err := parseFileToPackage(log, dir, path, pkg.Name, tokenFiles, file, pkg)
			if err != nil {
				log.Emit(metrics.Error(err), metrics.With("message", "Failed to parse file"), metrics.With("dir", dir), metrics.With("file", file.Name.Name), metrics.With("Package", pkg.Name))
				return nil, err
			}

			if err := res.loadImported(log); err != nil {
				log.Emit(metrics.Error(err), metrics.With("message", "Failed to load imported pacakges"), metrics.With("dir", dir), metrics.With("file", file.Name.Name), metrics.With("Package", pkg.Name))
				return nil, err
			}

			log.Emit(metrics.Info("Parsed Package File"), metrics.With("dir", dir), metrics.With("file", file.Name.Name), metrics.With("path", path), metrics.With("Package", pkg.Name))

			if owner, ok := packageDeclrs[pkg.Name]; ok {
				if strings.HasSuffix(tag, "_test") {
					owner.TestPackages = append(owner.TestPackages, res)
				} else {
					owner.Packages = append(owner.Packages, res)
				}

				packageDeclrs[res.Package] = owner
				continue
			}

			var testPkgs, codePkgs []PackageDeclaration

			if strings.HasSuffix(tag, "_test") {
				testPkgs = append(testPkgs, res)
			} else {
				codePkgs = append(codePkgs, res)
			}

			packageDeclrs[res.Package] = Package{
				Tag:          tag,
				Name:         res.Package,
				Path:         res.Path,
				FilePath:     filepath.Base(res.FilePath),
				BuildPkg:     buildPkg,
				Files:        pkgFiles,
				Packages:     codePkgs,
				TestPackages: testPkgs,
			}
		}

		if owner, ok := packageDeclrs[pkg.Name]; ok {
			owner.Files = pkgFiles
			packageDeclrs[pkg.Name] = owner
		}
	}

	var pkgs []Package
	for _, pkg := range packageDeclrs {
		pkgs = append(pkgs, pkg)
	}

	return pkgs, nil
}

// PackageWithBuildCtx parses the package directory which generates a series of ast with associated
// annotation for processing by using the golang token parser, it uses the build.Context to
// collected context details for the package but does not use it has a means to select the files to
// process. PackageWithBuildCtx processes all files in package directory. If you want one which takes
// into consideration build.Context fields using FilteredPackageWithBuildCtx.
func PackageWithBuildCtx(log metrics.Metrics, dir string, ctx build.Context) ([]Package, error) {
	tokenFiles := token.NewFileSet()
	packages, err := parser.ParseDir(tokenFiles, dir, nil, parser.ParseComments)
	if err != nil {
		log.Emit(metrics.Error(err), metrics.With("message", "Failed to parse directory"), metrics.With("dir", dir))
		return nil, err
	}

	packageDeclrs := make(map[string]Package)
	packageBuilds := make(map[string]*build.Package)

	for pkgTag, pkg := range packages {
		uniqueDir := fmt.Sprintf("%s#%s", dir, pkgTag)

		processedPackages.pl.Lock()
		res, ok := processedPackages.pkgs[uniqueDir]
		if ok {
			log.Emit(metrics.Info("Skipping package processing"), metrics.With("dir", dir))
			processedPackages.pl.Unlock()
			packageDeclrs[pkg.Name] = res
			continue
		}
		processedPackages.pl.Unlock()

		var pkgFiles []string

		for path, file := range pkg.Files {
			pkgFiles = append(pkgFiles, path)

			pathPkg := filepath.Dir(path)
			buildPkg, ok := packageBuilds[pathPkg]
			if !ok {
				buildPkg, err = ctx.ImportDir(pathPkg, 0)
				if err != nil {
					log.Emit(metrics.Errorf("Failed to retrieve build.Package"),
						metrics.With("file", path),
						metrics.With("dir", dir),
						metrics.With("file-dir", filepath.Dir(path)),
						metrics.With("error", err.Error()),
						metrics.With("mode", build.FindOnly))
				} else {
					packageBuilds[pathPkg] = buildPkg
					log.Emit(metrics.Info("Generated build.Package"),
						metrics.With("file", path),
						metrics.With("pkg", buildPkg),
						metrics.With("file-dir", filepath.Dir(path)),
						metrics.With("dir", dir),
						metrics.With("mode", build.FindOnly))
				}
			}

			res, err := parseFileToPackage(log, dir, path, pkg.Name, tokenFiles, file, pkg)
			if err != nil {
				log.Emit(metrics.Error(err), metrics.With("message", "Failed to parse file"), metrics.With("dir", dir), metrics.With("file", file.Name.Name), metrics.With("Package", pkg.Name))
				return nil, err
			}

			if owner, ok := packageDeclrs[pkg.Name]; ok {
				if strings.HasSuffix(pkgTag, "_test") {
					owner.TestPackages = append(owner.TestPackages, res)
				} else {
					owner.Packages = append(owner.Packages, res)
				}

				packageDeclrs[res.Package] = owner
				continue
			}

			var testPkgs, codePkgs []PackageDeclaration

			if strings.HasSuffix(pkgTag, "_test") {
				testPkgs = append(testPkgs, res)
			} else {
				codePkgs = append(codePkgs, res)
			}

			impPkg := Package{
				Name:         res.Package,
				FilePath:     path,
				Path:         res.Path,
				Tag:          pkgTag,
				BuildPkg:     buildPkg,
				Packages:     codePkgs,
				TestPackages: testPkgs,
			}

			packageDeclrs[pkg.Name] = impPkg
		}

		if owner, ok := packageDeclrs[pkg.Name]; ok {
			owner.Files = pkgFiles
			packageDeclrs[pkg.Name] = owner

			processedPackages.pl.Lock()
			processedPackages.pkgs[uniqueDir] = owner
			processedPackages.pl.Unlock()
		}
	}

	var pkgs []Package
	for _, pkg := range packageDeclrs {
		if err := pkg.loadImported(log); err != nil {
			log.Emit(metrics.Error(err), metrics.With("message", "Failed to load imported pacakges"), metrics.With("pkg", pkg.Path))
			return nil, err
		}

		pkgs = append(pkgs, pkg)
	}

	return pkgs, nil
}

// PackageFileWithBuildCtx parses the package from the provided file.
func PackageFileWithBuildCtx(log metrics.Metrics, path string, ctx build.Context) (Package, error) {
	dir := filepath.Dir(path)
	fName := filepath.Base(path)

	buildPkg, err := ctx.ImportDir(dir, 0)
	if err != nil {
		log.Emit(metrics.Errorf("Failed to retrieve build.Package"),
			metrics.With("file", path),
			metrics.With("dir", dir),
			metrics.With("error", err.Error()),
			metrics.With("mode", build.FindOnly))
	}

	log.Emit(metrics.Info("Generated build.Package"),
		metrics.With("file", path),
		metrics.With("dir", dir),
		metrics.With("pkg", buildPkg),
		metrics.With("mode", build.FindOnly))

	allowed := map[string]bool{
		fName: true,
	}

	filter := func(f os.FileInfo) bool {
		log.Emit(metrics.Info("Parse Filtering file"), metrics.With("incoming-file", f.Name()), metrics.With("allowed", allowed[f.Name()]))
		return allowed[f.Name()]
	}

	tokenFiles := token.NewFileSet()
	packages, err := parser.ParseDir(tokenFiles, path, filter, parser.ParseComments)
	if err != nil {
		log.Emit(metrics.Error(err), metrics.With("message", "Failed to parse file"), metrics.With("dir", dir), metrics.With("file", path))
		return Package{}, err
	}

	var pkg *ast.Package
	var pkgTag string

	pkgName := filepath.Base(filepath.Dir(path))
	for pkgTag, pkg = range packages {
		if pkg.Name != pkgName {
			continue
		}
		break
	}

	var pkgFiles []string

	for fpath, file := range pkg.Files {
		if fpath != path {
			continue
		}

		pkgFiles = append(pkgFiles, fpath)

		res, err := parseFileToPackage(log, dir, path, buildPkg.Name, tokenFiles, file, pkg)
		if err != nil {
			log.Emit(metrics.Error(err), metrics.With("message", "Failed to parse file"), metrics.With("dir", dir), metrics.With("file", file.Name.Name), metrics.With("Package", pkg.Name))
			return Package{}, err
		}

		if err := res.loadImported(log); err != nil {
			log.Emit(metrics.Error(err), metrics.With("message", "Failed to load imported pacakges"), metrics.With("dir", dir), metrics.With("file", file.Name.Name), metrics.With("Package", pkg.Name))
			return Package{}, err
		}

		var testPkgs, codePkgs []PackageDeclaration

		if strings.HasSuffix(pkgTag, "_test") {
			testPkgs = append(testPkgs, res)
		} else {
			codePkgs = append(codePkgs, res)
		}

		return Package{
			BuildPkg:     buildPkg,
			Tag:          pkgTag,
			Files:        pkgFiles,
			Path:         res.Path,
			Name:         res.Package,
			FilePath:     res.FilePath,
			Packages:     codePkgs,
			TestPackages: testPkgs,
		}, nil
	}

	return Package{}, ErrPackageParseFailed
}

func parseFileToPackage(log metrics.Metrics, dir string, path string, pkgName string, tokenFiles *token.FileSet, file *ast.File, pkgAstObj *ast.Package) (PackageDeclaration, error) {
	var packageDeclr PackageDeclaration

	{
		pkgSource, _ := readSource(path)

		packageDeclr.Package = pkgName
		packageDeclr.FilePath = path
		packageDeclr.Source = string(pkgSource)
		packageDeclr.Imports = make(map[string]ImportDeclaration, 0)
		packageDeclr.ObjectFunc = make(map[*ast.Object][]FuncDeclaration, 0)

		if file.Doc != nil {
			for _, comment := range file.Doc.List {
				packageDeclr.Comments = append(packageDeclr.Comments, comment.Text)
			}
		}

		for _, imp := range file.Imports {
			beginPosition, endPosition := tokenFiles.Position(imp.Pos()), tokenFiles.Position(imp.End())
			positionLength := endPosition.Offset - beginPosition.Offset
			source, err := readSourceIn(beginPosition.Filename, int64(beginPosition.Offset), positionLength)

			if err != nil {
				return packageDeclr, err
			}

			var pkgName string

			if imp.Name != nil {
				pkgName = strings.Replace(imp.Name.Name, "/", "", -1)
			} else {
				pkgName = strings.Replace(filepath.Base(imp.Path.Value), "\"", "", -1)
			}

			if pkgNm, perr := strconv.Unquote(pkgName); perr == nil {
				pkgName = pkgNm
			}

			impPkgPath, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				impPkgPath = imp.Path.Value
			}

			var comment string
			if imp.Comment != nil {
				comment = imp.Comment.Text()
			}

			var internal bool
			if _, err := relativeToSrc(filepath.Join(goSrcPath, impPkgPath)); err != nil {
				internal = true
			}

			packageDeclr.Imports[pkgName] = ImportDeclaration{
				Comments:    comment,
				Name:        pkgName,
				Path:        impPkgPath,
				InternalPkg: internal,
				Source:      string(source),
			}
		}

		if relPath, err := relativeToSrc(path); err == nil {
			packageDeclr.Path = filepath.Dir(relPath)
			packageDeclr.File = filepath.Base(relPath)
		}

		if runtime.GOOS == "windows" {
			packageDeclr.Path = filepath.ToSlash(packageDeclr.Path)
			packageDeclr.File = filepath.ToSlash(packageDeclr.File)
			packageDeclr.FilePath = filepath.ToSlash(packageDeclr.FilePath)
		}

		if file.Doc != nil {
			annotationRead := ReadAnnotationsFromCommentry(bytes.NewBufferString(file.Doc.Text()))

			log.Emit(metrics.Info("Annotations in Package comments"),
				metrics.With("dir", dir),
				metrics.With("annotations", len(annotationRead)),
				metrics.With("file", file.Name.Name))

			packageDeclr.Annotations = append(packageDeclr.Annotations, annotationRead...)
		}

		// Collect and categorize annotations in types and their fields.
	declrLoop:
		for _, declr := range file.Decls {
			tokenFile := tokenFiles.File(declr.Pos())
			beginPosition, endPosition := tokenFile.Position(declr.Pos()), tokenFile.Position(declr.End())
			beginOffset := beginPosition.Offset
			endOffset := endPosition.Offset

			positionLength := (endOffset - beginOffset)
			source, err := readSourceIn(tokenFile.Name(), int64(beginOffset), positionLength)
			if err != nil {
				return packageDeclr, err
			}

			switch rdeclr := declr.(type) {
			case *ast.FuncDecl:
				var comment string

				if rdeclr.Doc != nil {
					comment = rdeclr.Doc.Text()
				}

				var annotations []AnnotationDeclaration
				associations := make(map[string]AnnotationAssociationDeclaration, 0)

				if rdeclr.Doc != nil {
					annotationRead := ReadAnnotationsFromCommentry(bytes.NewBufferString(rdeclr.Doc.Text()))

					for _, item := range annotationRead {
						log.Emit(metrics.Info("Annotation in Function Decleration comment"), metrics.With("dir", dir), metrics.With("annotation", item.Name))

						switch item.Name {
						case "associates":
							log.Emit(
								metrics.Info("Association found"),
								metrics.With("dir", dir),
								metrics.With("association", item.Arguments),
							)

							if len(item.Arguments) >= 3 {
								associations[item.Arguments[0]] = AnnotationAssociationDeclaration{
									Record:     item,
									Template:   item.Template,
									Action:     item.Arguments[1],
									TypeName:   item.Arguments[2],
									Annotation: strings.TrimPrefix(item.Arguments[0], "@"),
								}
							} else {
								log.Emit(metrics.Error(errors.New("Association Annotation in Declaration is incomplete: Expects 3 elements")), metrics.With("dir", dir), metrics.With("association", item.Arguments))
							}
						default:
							annotations = append(annotations, item)
						}
					}
				}

				var defFunc FuncDeclaration

				defFunc.Comments = comment
				defFunc.Source = string(source)
				defFunc.TypeDeclr = declr
				defFunc.FuncDeclr = rdeclr
				defFunc.Type = rdeclr.Type
				defFunc.Position = rdeclr.Pos()
				defFunc.Path = packageDeclr.Path
				defFunc.File = packageDeclr.File
				defFunc.Declr = &packageDeclr
				defFunc.FuncName = rdeclr.Name.Name
				defFunc.Length = positionLength
				defFunc.From = beginPosition.Offset
				defFunc.Package = packageDeclr.Package
				defFunc.FilePath = packageDeclr.FilePath
				defFunc.Annotations = annotations
				defFunc.Associations = associations
				defFunc.Exported = unicode.IsUpper(rune(rdeclr.Name.Name[0]))

				if rdeclr.Type != nil {
					defFunc.Returns = rdeclr.Type.Results
					defFunc.Arguments = rdeclr.Type.Params
				}

				if rdeclr.Recv != nil {
					defFunc.FuncType = rdeclr.Recv

					nameIdent := rdeclr.Recv.List[0]
					var receiverNameType *ast.Ident

					switch nmi := nameIdent.Type.(type) {
					case *ast.Ident:
						receiverNameType = nmi
					case *ast.StarExpr:
						receiverNameType = nmi.X.(*ast.Ident)
						defFunc.RecieverPointer = nmi
					}

					defFunc.Reciever = receiverNameType.Obj
					defFunc.RecieverIdent = receiverNameType
					defFunc.RecieverName = receiverNameType.Name

					if rems, ok := packageDeclr.ObjectFunc[receiverNameType.Obj]; ok {
						rems = append(rems, defFunc)
						packageDeclr.ObjectFunc[receiverNameType.Obj] = rems
					} else {
						packageDeclr.ObjectFunc[receiverNameType.Obj] = []FuncDeclaration{defFunc}
					}

					continue declrLoop
				}

				packageDeclr.Functions = append(packageDeclr.Functions, defFunc)
				continue declrLoop

			case *ast.GenDecl:
				var comment string

				if rdeclr.Doc != nil {
					comment = rdeclr.Doc.Text()
				}

				var annotations []AnnotationDeclaration

				associations := make(map[string]AnnotationAssociationDeclaration, 0)

				if rdeclr.Doc != nil {
					annotationRead := ReadAnnotationsFromCommentry(bytes.NewBufferString(rdeclr.Doc.Text()))

					for _, item := range annotationRead {
						log.Emit(metrics.Info("Annotation in Decleration comment"),
							metrics.With("dir", dir),
							metrics.With("annotation", item.Name))

						switch item.Name {
						case "associates":
							log.Emit(
								metrics.Info("Association found"),
								metrics.With("dir", dir),
								metrics.With("association", item.Arguments),
							)

							if len(item.Arguments) >= 3 {
								associations[item.Arguments[0]] = AnnotationAssociationDeclaration{
									Record:     item,
									Template:   item.Template,
									Action:     item.Arguments[1],
									TypeName:   item.Arguments[2],
									Annotation: strings.TrimPrefix(item.Arguments[0], "@"),
								}
							} else {
								log.Emit(metrics.Error(errors.New("Association Annotation in Declaration is incomplete: Expects 3 elements")), metrics.With("dir", dir), metrics.With("association", item.Arguments))
							}
						default:
							annotations = append(annotations, item)
						}
					}
				}

				for _, spec := range rdeclr.Specs {
					switch obj := spec.(type) {
					case *ast.ValueSpec:
						// Handles variable declaration
						// i.e Spec:
						// &ast.ValueSpec{Doc:(*ast.CommentGroup)(nil), Names:[]*ast.Ident{(*ast.Ident)(0xc4200e4a00)}, Type:ast.Expr(nil), Values:[]ast.Expr{(*ast.BasicLit)(0xc4200e4a20)}, Comment:(*ast.CommentGroup)(nil)}
						// &ast.ValueSpec{Doc:(*ast.CommentGroup)(nil), Names:[]*ast.Ident{(*ast.Ident)(0xc4200e4a40)}, Type:(*ast.Ident)(0xc4200e4a60), Values:[]ast.Expr(nil), Comment:(*ast.CommentGroup)(nil)}
						packageDeclr.Variables = append(packageDeclr.Variables, VariableDeclaration{
							Object:       obj,
							Annotations:  annotations,
							Associations: associations,
							GenObj:       rdeclr,
							Source:       string(source),
							Comments:     comment,
							Declr:        &packageDeclr,
							File:         packageDeclr.File,
							Package:      packageDeclr.Package,
							Path:         packageDeclr.Path,
							FilePath:     packageDeclr.FilePath,
							From:         beginPosition.Offset,
							Length:       positionLength,
						})

					case *ast.TypeSpec:

						switch robj := obj.Type.(type) {
						case *ast.StructType:

							log.Emit(metrics.Info("Annotation in Decleration"),
								metrics.With("Type", "Struct"),
								metrics.With("Annotations", len(annotations)),
								metrics.With("StructName", obj.Name.Name))

							packageDeclr.Structs = append(packageDeclr.Structs, StructDeclaration{
								Object:       obj,
								Struct:       robj,
								Annotations:  annotations,
								Associations: associations,
								GenObj:       rdeclr,
								Source:       string(source),
								Comments:     comment,
								Declr:        &packageDeclr,
								File:         packageDeclr.File,
								Package:      packageDeclr.Package,
								Path:         packageDeclr.Path,
								FilePath:     packageDeclr.FilePath,
								From:         beginPosition.Offset,
								Length:       positionLength,
							})

						case *ast.InterfaceType:
							log.Emit(metrics.Info("Annotation in Decleration"),
								metrics.With("Type", "Interface"),
								metrics.With("Annotations", len(annotations)),
								metrics.With("StructName", obj.Name.Name))

							packageDeclr.Interfaces = append(packageDeclr.Interfaces, InterfaceDeclaration{
								Object:       obj,
								Interface:    robj,
								GenObj:       rdeclr,
								Comments:     comment,
								Annotations:  annotations,
								Associations: associations,
								Declr:        &packageDeclr,
								Source:       string(source),
								File:         packageDeclr.File,
								Package:      packageDeclr.Package,
								Path:         packageDeclr.Path,
								FilePath:     packageDeclr.FilePath,
								From:         beginPosition.Offset,
								Length:       positionLength,
							})

						default:
							log.Emit(metrics.Info("Annotation in Decleration"),
								metrics.With("Type", "OtherType"),
								metrics.With("Marker", "NonStruct/NonInterface:Type"),
								metrics.With("Annotations", len(annotations)),
								metrics.With("StructName", obj.Name.Name))

							packageDeclr.Types = append(packageDeclr.Types, TypeDeclaration{
								Object:       obj,
								GenObj:       rdeclr,
								Annotations:  annotations,
								Comments:     comment,
								Associations: associations,
								Declr:        &packageDeclr,
								Source:       string(source),
								File:         packageDeclr.File,
								Package:      packageDeclr.Package,
								Path:         packageDeclr.Path,
								FilePath:     packageDeclr.FilePath,
								From:         beginPosition.Offset,
								Length:       positionLength,
							})
						}

					case *ast.ImportSpec:
						// Do Nothing.
					}
				}

			case *ast.BadDecl:
				// Do Nothing.
			}
		}

	}

	return packageDeclr, nil
}

func relativeToSrc(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", err
	}

	return filepath.Rel(goSrcPath, path)
}

//===========================================================================================================

// SimplyParse takes the provided packages parsing all internals declarations with the appropriate generators suited to the type and annotations.
// Relies on SimpleParsePackage.
func SimplyParse(toDir string, log metrics.Metrics, provider *AnnotationRegistry, doFileOverwrite bool, pkgDeclrs ...Package) error {
	for _, pkg := range pkgDeclrs {
		if err := SimplyParsePackage(toDir, log, provider, doFileOverwrite, pkg); err != nil {
			return err
		}
	}

	return nil
}

// Parse takes the provided packages parsing all internals declarations with the appropriate generators suited to the type and annotations.
// Relies on ParsePackage.
func Parse(toDir string, log metrics.Metrics, provider *AnnotationRegistry, doFileOverwrite bool, pkgDeclrs ...Package) error {
	for _, pkg := range pkgDeclrs {
		if err := ParsePackage(toDir, log, provider, doFileOverwrite, pkg); err != nil {
			return err
		}
	}

	return nil
}

// WriteDirectives defines a function which houses the logic to write WriteDirective into file system.
func WriteDirectives(log metrics.Metrics, toDir string, doFileOverwrite bool, wds ...gen.WriteDirective) error {
	for _, wd := range wds {
		if err := WriteDirective(log, toDir, doFileOverwrite, wd); err != nil {
			return err
		}
	}

	return nil
}

// SimpleWriteDirectives defines a function which houses the logic to write WriteDirective into file system.
func SimpleWriteDirectives(toDir string, doFileOverwrite bool, wds ...gen.WriteDirective) error {
	for _, wd := range wds {
		if err := SimpleWriteDirective(toDir, doFileOverwrite, wd); err != nil {
			return err
		}
	}

	return nil
}

// SimpleWriteDirective defines a function which houses the logic to write WriteDirective into file system.
func SimpleWriteDirective(toDir string, doFileOverwrite bool, item gen.WriteDirective) error {
	if filepath.IsAbs(item.Dir) {
		return fmt.Errorf("gen.WriteDirectiveError: Expected relative Dir path not absolute: %+q", item.Dir)
	}

	if item.Before != nil {
		if err := item.Before(); err != nil {
			return err
		}
	}

	namedFileDir := toDir
	baseDir := toDir

	if item.Dir != "" {
		namedFileDir = filepath.Join(toDir, item.Dir)
	}

	if filepath.IsAbs(baseDir) {
		baseDir = filepath.Base(baseDir)
	}

	if namedFileDir != "" {
		if _, err := os.Stat(namedFileDir); err != nil {
			err = os.MkdirAll(namedFileDir, 0700)
			if err != nil && err != os.ErrExist {
				err = fmt.Errorf("IOError: Unable to create directory: %+q", err)
				return err
			}

			fmt.Printf("Creating directory %q\n", filepath.Join(baseDir, item.Dir))
		}
	}

	if item.Writer == nil {
		return nil
	}

	if item.FileName == "" {
		err := fmt.Errorf("WriteDirective has no filename value attached")
		return err
	}

	namedFile := filepath.Join(namedFileDir, item.FileName)

	fileStat, err := os.Stat(namedFile)
	if err == nil && !fileStat.IsDir() && item.DontOverride && !doFileOverwrite {
		return err
	}

	newFile, err := os.Create(namedFile)
	if err != nil {
		return err
	}

	fmt.Printf("Creating new file %q\n", filepath.Join(baseDir, item.Dir, item.FileName))

	defer newFile.Close()

	_, err = item.Writer.WriteTo(newFile)
	if err != nil && err != io.EOF {
		err = fmt.Errorf("IOError: Unable to write content to file: %+q", err)
		return err
	}

	if item.After == nil {
		return nil
	}

	return item.After()
}

// WriteDirective defines a function which houses the logic to write WriteDirective into file system.
func WriteDirective(log metrics.Metrics, toDir string, doFileOverwrite bool, item gen.WriteDirective) error {
	if item.Before != nil {
		if err := item.Before(); err != nil {
			return err
		}
	}

	log.Emit(metrics.Info("Execute WriteDirective"),
		metrics.With("overwrite", item.DontOverride),
		metrics.With("action", actions.MkDirectory{
			Dir:     item.Dir,
			RootDir: toDir,
			Mode:    0700,
		}))

	if filepath.IsAbs(item.Dir) {
		err := fmt.Errorf("gen.WriteDirectiveError: Expected relative Dir path not absolute: %+q", item.Dir)
		log.Emit(metrics.Error(err), metrics.With("File", item.FileName), metrics.With("Overwrite", item.DontOverride), metrics.With("Dir", item.Dir))
		return err
	}

	namedFileDir := toDir
	if item.Dir != "" {
		namedFileDir = filepath.Join(toDir, item.Dir)
	}

	if namedFileDir != "" {
		if err := os.MkdirAll(namedFileDir, 0700); err != nil && err != os.ErrExist {
			err = fmt.Errorf("IOError: Unable to create directory: %+q", err)
			log.Emit(metrics.Error(err),
				metrics.With("overwrite", item.DontOverride),
				metrics.With("action", events.DirCreated{
					Error: err,
					Action: actions.MkDirectory{
						Dir:     item.Dir,
						RootDir: toDir,
						Mode:    0700,
					},
				}))
			return err
		}
	}

	log.Emit(metrics.Info("Resolved WriteDirective"),
		metrics.With("op", "mkdir"),
		metrics.With("action", events.DirCreated{
			Action: actions.MkDirectory{
				Dir:     item.Dir,
				RootDir: toDir,
				Mode:    0700,
			},
		}))

	if item.Writer == nil {
		log.Emit(metrics.Info("Resolved WriteDirective"), metrics.With("File", item.FileName), metrics.With("Overwrite", item.DontOverride), metrics.With("Dir", item.Dir))
		return nil
	}

	if item.FileName == "" {
		err := fmt.Errorf("WriteDirective has no filename value attached")
		log.Emit(metrics.Error(err), metrics.With("File", item.FileName), metrics.With("Overwrite", item.DontOverride), metrics.With("Dir", item.Dir))
		return err
	}

	namedFile := filepath.Join(namedFileDir, item.FileName)

	fileStat, err := os.Stat(namedFile)
	if err == nil && !fileStat.IsDir() && item.DontOverride && !doFileOverwrite {
		log.Emit(metrics.Info("File overwrite not aloud"), metrics.With("File", item.FileName),
			metrics.With("Overwrite", item.DontOverride),
			metrics.With("Dir", item.Dir),
			metrics.With("DestinationDir", namedFileDir),
			metrics.With("DestinationFile", namedFile))
		return err
	}

	newFile, err := os.Create(namedFile)
	if err != nil {
		log.Emit(metrics.Error(err), metrics.With("File", item.FileName), metrics.With("Overwrite", item.DontOverride), metrics.With("Dir", item.Dir),
			metrics.With("DestinationDir", namedFileDir),
			metrics.With("DestinationFile", namedFile))
		return err
	}

	defer newFile.Close()

	written, err := item.Writer.WriteTo(newFile)
	if err != nil && err != io.EOF {
		err = fmt.Errorf("IOError: Unable to write content to file: %+q", err)
		log.Emit(metrics.Error(err), metrics.With("File", item.FileName), metrics.With("Overwrite", item.DontOverride), metrics.With("Dir", item.Dir),
			metrics.With("DestinationDir", namedFileDir),
			metrics.With("DestinationFile", namedFile))
		return err
	}

	log.Emit(metrics.Info("Resolved WriteDirective"),
		metrics.With("op", "writefile"),
		metrics.With("action", events.FileCreated{
			Written: written,
			Action: actions.CreateFile{
				RootDir:  toDir,
				Dir:      item.Dir,
				FileName: item.FileName,
				Mode:     0700,
			},
		}))

	if item.After == nil {
		return nil
	}

	return item.After()
}

// ParsePackage takes the provided package declrations parsing all internals with the appropriate generators suited to the type and annotations.
// Provided toDir must be a absolute path.
func ParsePackage(toDir string, log metrics.Metrics, provider *AnnotationRegistry, doFileOverwrite bool, pkgDeclrs Package) error {
	log.Emit(metrics.Info("Begin ParsePackage"), metrics.With("toDir", toDir),
		metrics.With("overwriter-file", doFileOverwrite),
		metrics.With("package", pkgDeclrs.Path))

	if !filepath.IsAbs(toDir) {
		return errors.New("Destination path must be a absolute path directory")
	}

	toSrcPath, err := srcpath.RelativeToSrc(toDir)
	if err != nil {
		return fmt.Errorf("Destination path is not within current GOPATH: %+q", err.Error())
	}

	for _, pkg := range pkgDeclrs.Packages {
		log.Emit(metrics.Info("ParsePackage: Parse PackageDeclaration"),
			metrics.With("toDir", toDir), metrics.With("overwriter-file", doFileOverwrite),
			metrics.With("package", pkg.Package),
			metrics.With("From", pkg.FilePath))

		wdrs, err := provider.ParseDeclr(pkgDeclrs, pkg, toSrcPath)
		if err != nil {
			log.Emit(metrics.Error(fmt.Errorf("ParseFailure: Package %q", pkg.Package)),
				metrics.With("error", err.Error()), metrics.With("package", pkg.Package))
			return err
		}

		log.Emit(metrics.Info("ParseSuccess"), metrics.With("From", pkg.FilePath), metrics.With("package", pkg.Package), metrics.With("Directives", len(wdrs)))

		for _, wd := range wdrs {
			if err := WriteDirective(log, toDir, doFileOverwrite, wd.WriteDirective); err != nil {
				log.Emit(metrics.Error(err), metrics.With("annotation", wd.Annotation),
					metrics.With("dir", toDir),
					metrics.With("package", pkg.Package),
					metrics.With("file", pkg.File))
				return err
			}

			log.Emit(metrics.Info("Annotation Resolved"), metrics.With("annotation", wd.Annotation),
				metrics.With("dir", toDir),
				metrics.With("package", pkg.Package),
				metrics.With("file", pkg.File))
		}

	}

	return nil
}

// SimplyParsePackage takes the provided package declrations parsing all internals with the appropriate generators suited to the type and annotations.
// Provided toDir must be a absolute path.
func SimplyParsePackage(toDir string, log metrics.Metrics, provider *AnnotationRegistry, doFileOverwrite bool, pkgDeclrs Package) error {
	log.Emit(metrics.Info("Begin ParsePackage"), metrics.With("toDir", toDir),
		metrics.With("overwriter-file", doFileOverwrite),
		metrics.With("package", pkgDeclrs.Path))

	if !filepath.IsAbs(toDir) {
		return errors.New("Destination path must be a absolute path directory")
	}

	toSrcPath, err := srcpath.RelativeToSrc(toDir)
	if err != nil {
		return fmt.Errorf("Destination path is not within current GOPATH: %+q", err.Error())
	}

	for _, pkg := range pkgDeclrs.Packages {
		log.Emit(metrics.Info("ParsePackage: Parse PackageDeclaration"),
			metrics.With("toDir", toDir), metrics.With("overwriter-file", doFileOverwrite),
			metrics.With("package", pkg.Package),
			metrics.With("From", pkg.FilePath))

		wdrs, err := provider.ParseDeclr(pkgDeclrs, pkg, toSrcPath)
		if err != nil {
			log.Emit(metrics.Error(fmt.Errorf("ParseFailure: Package %q", pkg.Package)),
				metrics.With("error", err.Error()), metrics.With("package", pkg.Package))
			return err
		}

		log.Emit(metrics.Info("ParseSuccess"), metrics.With("From", pkg.FilePath), metrics.With("package", pkg.Package), metrics.With("Directives", len(wdrs)))

		for _, wd := range wdrs {
			if err := SimpleWriteDirective(toDir, doFileOverwrite, wd.WriteDirective); err != nil {
				log.Emit(metrics.Error(err), metrics.With("annotation", wd.Annotation),
					metrics.With("dir", toDir),
					metrics.With("package", pkg.Package),
					metrics.With("file", pkg.File))
				return err
			}

			log.Emit(metrics.Info("Annotation Resolved"), metrics.With("annotation", wd.Annotation),
				metrics.With("dir", toDir),
				metrics.With("package", pkg.Package),
				metrics.With("file", pkg.File))
		}

	}

	return nil
}

//===========================================================================================================

// WhichPackage is an utility function which returns the appropriate package name to use
// if a toDir is provided as destination.
func WhichPackage(toDir string, pkg Package) string {
	if toDir != "" && toDir != "./" && toDir != "." {
		return strings.ToLower(filepath.Base(toDir))
	}

	return pkg.Name
}

//===========================================================================================================
