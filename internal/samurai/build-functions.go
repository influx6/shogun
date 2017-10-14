package samurai

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"go/doc"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"

	"github.com/influx6/faux/exec"
	"github.com/influx6/faux/fmtwriter"
	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/vfiles"
	"github.com/influx6/gobuild/build"
	"github.com/influx6/gobuild/srcpath"
	"github.com/influx6/moz/ast"
	"github.com/influx6/moz/gen"
	"github.com/influx6/shogun/internal"
	"github.com/influx6/shogun/templates"
)

var (
	ignoreAddition = ".shogun"
	goosRuntime    = runtime.GOOS
	binNameReg     = regexp.MustCompile("\\W+")
	packageReg     = regexp.MustCompile(`package \w+`)
)

// BuildFunctions holds all build directives from processed packages.
type BuildFunctions struct {
	Dir  string
	Main BuildList
	Subs map[string]BuildList
}

// BuildPackage builds a shogun binarie commandline files for giving directory and 1 level directory.
func BuildPackage(vlog metrics.Metrics, events metrics.Metrics, dir string, cmdDir string, cuDir string, binaryPath string, skipBuild bool, dontCombineRoot bool, ctx build.Context) (BuildFunctions, error) {
	var list BuildFunctions
	list.Subs = make(map[string]BuildList)

	if err := vfiles.WalkDirSurface(dir, func(rel string, abs string, info os.FileInfo) error {
		if !info.IsDir() {
			return nil
		}

		res, err2 := BuildPackageForDir(vlog, events, abs, cmdDir, cuDir, binaryPath, skipBuild, map[string]BuildList{}, ctx)
		if err2 != nil {
			if err2 == ErrSkipDir {
				return nil
			}

			events.Emit(metrics.Error(err2).With("dir", abs).With("binary_path", binaryPath))
			return err2
		}

		res.RelPath = rel
		list.Subs[res.Path] = res
		return nil
	}); err != nil {
		events.Emit(metrics.Error(err).With("dir", dir))
		return list, err
	}

	var combined map[string]BuildList

	if !dontCombineRoot {
		combined = list.Subs
	}

	var err error
	list.Main, err = BuildPackageForDir(vlog, events, dir, cmdDir, cuDir, binaryPath, skipBuild, combined, ctx)
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", dir).With("binary_path", binaryPath))
		return list, err
	}

	return list, nil
}

// BuildList holds a procssed package list of write directives.
type BuildList struct {
	Hash            string
	Path            string
	RelPath         string
	FromPackage     string
	FromPackageName string
	PkgPath         string
	BasePkgPath     string
	BinaryName      string
	CleanBinaryName string
	Desc            string
	ExecBinaryName  string
	List            []gen.WriteDirective
	Functions       []internal.PackageFunctions
}

// HasGoogleImports returns true/false if any part of the function uses faux context.
func (pn BuildList) HasGoogleImports() bool {
	for _, item := range pn.Functions {
		if item.HasGoogleImports() {
			return true
		}
	}

	return false
}

// HasFauxImports returns true/false if any part of the function uses faux context.
func (pn BuildList) HasFauxImports() bool {
	for _, item := range pn.Functions {
		if item.HasFauxImports() {
			return true
		}
	}

	return false
}

// BuildPackageForDir generates needed package files for creating new function based executable binaries.
func BuildPackageForDir(vlog metrics.Metrics, events metrics.Metrics, dir string, cmd string, currentDir string, binaryPath string, skipBuild bool, subs map[string]BuildList, ctx build.Context) (BuildList, error) {
	if subs == nil {
		subs = make(map[string]BuildList)
	}

	var list BuildList
	list.Path = dir

	pkgs, err := ast.FilteredPackageWithBuildCtx(vlog, dir, ctx)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			return list, ErrSkipDir
		}

		return list, err
	}

	if len(pkgs) == 0 {
		return list, ErrSkipDir
	}

	pkgItem := pkgs[0]
	if pkgItem.HasAnnotation("@shogunIgnore") {
		return list, ErrSkipDir
	}

	pkgHash, err := generateHash(pkgItem.Files)
	if err != nil {
		return list, err
	}

	var binaryName, binaryDesc, binaryExeName string
	if binAnnons := pkgItem.AnnotationsFor("@binaryName"); len(binAnnons) != 0 {
		if len(binAnnons[0].Arguments) == 0 {
			err2 := fmt.Errorf("InvalidBinaryName(File: %q): expected format @binaryName(name => NAME)", pkgItem.FilePath)
			return list, err2
		}

		binaryName = strings.ToLower(binAnnons[0].Param("name"))

		desc := binAnnons[0].Param("desc")
		if desc != "" && !strings.HasSuffix(desc, ".") {
			desc += "."
			binaryDesc = doc.Synopsis(desc)
		}

	} else {
		binaryName = pkgItem.Name
	}

	if binaryDesc == "" {
		binaryDesc = haiku()
	}

	binaryExeName = binaryName
	if goosRuntime == "windows" {
		binaryExeName = fmt.Sprintf("%s.exec", binaryName)
	}

	list.Desc = binaryDesc
	list.BinaryName = binaryName
	list.CleanBinaryName = toPackageName(binaryName)
	list.ExecBinaryName = binaryExeName
	list.FromPackage = pkgItem.Path
	list.FromPackageName = pkgItem.Name

	packageBinaryPath := filepath.Join(cmd, binaryName)
	packageBinaryFilePath := filepath.Join(packageBinaryPath, "pkg")
	totalPackageFilePath := filepath.Join(currentDir, packageBinaryFilePath)
	totalPackagePath, err := srcpath.RelativeToSrc(totalPackageFilePath)
	if err != nil {
		return list, fmt.Errorf("Expected package should be located in GOPATH/src: %+q", err)
	}

	var fnPkg internal.PackageFunctions
	fnPkg.Desc = binaryDesc
	fnPkg.Name = pkgItem.Name
	fnPkg.Path = pkgItem.Path
	fnPkg.BinaryName = binaryName
	fnPkg.FilePath = pkgItem.FilePath
	fnPkg.Hash = string(pkgHash)

	for _, declr := range pkgItem.Packages {
		// Retrieve function list if we are not to ingore file declr.
		if !declr.HasAnnotation("@shogunIgnoreFunctions") {
			fnsList, err := pullFunctionFromDeclr(pkgItem, &declr)
			if err != nil {
				return list, err
			}

			fnPkg.List = append(fnPkg.List, fnsList...)
		}

		source := strings.Replace(declr.Source, strings.Join(declr.Comments, "\n"), "", -1)
		packageIndex := strings.Index(source, "package")
		packagePart := packageReg.FindString(source)

		source = source[packageIndex:]
		source = strings.TrimSpace(strings.Replace(source, packagePart, "", 1))

		list.List = append(list.List, gen.WriteDirective{
			FileName: filepath.Base(declr.FilePath),
			Dir:      packageBinaryFilePath,
			Writer: gen.SourceTextWithName(
				"shogun:src-pkg-content",
				string(templates.Must("shogun-src-pkg-content.tml")),
				template.FuncMap{},
				struct {
					Source string
				}{
					Source: source,
				},
			),
		})
	}

	fnPkg.MaxNameLen = maxName(fnPkg)
	list.Functions = append(list.Functions, fnPkg)

	list.Hash = string(pkgHash)
	list.PkgPath = totalPackagePath
	list.BasePkgPath = packageBinaryPath

	list.List = append(list.List, gen.WriteDirective{
		FileName: fmt.Sprintf("pkg_%s.go", binaryFileName(binaryName)),
		Dir:      packageBinaryFilePath,
		Writer: fmtwriter.NewWith(vlog, gen.SourceTextWithName(
			"shogun:src-pkg",
			string(templates.Must("shogun-src-pkg.tml")),
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
			struct {
				BinaryName string
				Subs       map[string]BuildList
				Main       BuildList
			}{
				BinaryName: binaryName,
				Main:       list,
				Subs:       subs,
			},
		), true, true),
	})

	list.List = append(list.List, gen.WriteDirective{
		FileName: ".hashfile",
		Dir:      packageBinaryPath,
		Writer: gen.SourceTextWithName(
			"shogun:src-pkg-hash",
			string(templates.Must("shogun-src-pkg-hash.tml")),
			template.FuncMap{},
			struct {
				Hash string
			}{
				Hash: string(pkgHash),
			},
		),
	})

	var helpFormat bytes.Buffer

	formatMaker := gen.SourceTextWithName(
		"shogun-pkg-inbin-list",
		string(templates.Must("shogun-pkg-inbin-list.tml")),
		template.FuncMap{},
		struct {
			Main BuildList
			Subs map[string]BuildList
		}{
			Main: list,
			Subs: subs,
		},
	)

	if _, err := formatMaker.WriteTo(&helpFormat); err != nil {
		return list, fmt.Errorf("Failed to generate binary %q help message: %+q", binaryName, err)
	}

	list.List = append(list.List, gen.WriteDirective{
		FileName: "main.go",
		Dir:      packageBinaryPath,
		Writer: fmtwriter.NewWith(vlog, gen.SourceTextWithName(
			"shogun:src-pkg-main",
			string(templates.Must("shogun-src-pkg-main.tml")),
			template.FuncMap{},
			struct {
				Main               BuildList
				HelpFormat         string
				CustomHelpTemplate string
				BinaryName         string
				MainPackage        string
				Subs               map[string]BuildList
			}{
				Subs:               subs,
				Main:               list,
				BinaryName:         binaryName,
				MainPackage:        totalPackagePath,
				HelpFormat:         helpFormat.String(),
				CustomHelpTemplate: string(templates.Must("shogun-src-pkg-help-format.tml")),
			},
		), true, true),
		After: func() error {
			if skipBuild {
				return nil
			}

			fmt.Printf("----------------------------------------\n")
			fmt.Printf("Building binary for shogunate: %q\n", binaryName)

			var resp bytes.Buffer
			binCmd := exec.New(
				exec.Async(),
				exec.Err(&resp),
				exec.Command("go build -x -o %s %s",
					filepath.Join(binaryPath, binaryExeName),
					filepath.Join(packageBinaryPath, "main.go"),
				),
			)

			if err := binCmd.Exec(context.Background(), events); err != nil {
				fmt.Println(resp.String())
				fmt.Printf("Building binary for shogun %q failed\n", binaryName)
				return err
			}

			fmt.Printf("Built binary for shogun %q into %q\n", binaryName, binaryPath)

			fmt.Printf("Cleaning up shogun binary build files... %q\n", binaryName)
			if err := os.RemoveAll(filepath.Join(dir, packageBinaryPath)); err != nil {
				fmt.Printf("Failed to properly cleanup build files %q\n\n", binaryName)
				return err
			}

			fmt.Printf("Shogun %q build ready\n\n", binaryName)
			return nil
		},
	})

	return list, nil
}

func toPackageName(name string) string {
	return strings.ToLower(binNameReg.ReplaceAllString(name, ""))
}

func binaryFileName(name string) string {
	name = strings.Replace(name, "-", "_", -1)
	return name
}

func binHash(nlog metrics.Metrics, binPath string) (string, error) {
	var response bytes.Buffer

	if err := exec.New(exec.Command("%s hash", binPath), exec.Async(), exec.Output(&response)).Exec(context.Background(), nlog); err != nil {
		return "", err
	}

	return strings.TrimSpace(response.String()), nil
}

func generateHash(files []string) (string, error) {
	var hashes []byte

	for _, file := range files {
		hash, err := generateFileHash(file)
		if err != nil {
			return "", err
		}

		hashes = append(hashes, []byte(hash)...)
	}

	return base64.StdEncoding.EncodeToString(hashes), nil
}

func generateFileHash(file string) (string, error) {
	hasher := sha1.New()
	fl, err := os.Open(file)
	if err != nil {
		return "", err
	}

	defer fl.Close()

	_, err = io.Copy(hasher, fl)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
