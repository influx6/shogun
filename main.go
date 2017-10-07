package main

//go:generate go generate ./templates/...

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"go/doc"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/influx6/faux/exec"
	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/metrics/custom"
	"github.com/influx6/faux/vfiles"
	"github.com/influx6/gobuild/build"
	"github.com/influx6/gobuild/srcpath"
	"github.com/influx6/moz/ast"
	"github.com/influx6/moz/gen"
	"github.com/influx6/shogun/internal"
	"github.com/influx6/shogun/templates"
	"github.com/minio/cli"
)

// vars
var (
	Version          = "0.1"
	nolog            = metrics.New()
	shogunateDirName = "katanas"
	ignoreAddition   = ".shogun"
	goosRuntime      = runtime.GOOS
	events           = metrics.New(custom.StackDisplay(os.Stdout))
	packageReg       = regexp.MustCompile(`package \w+`)

	helpTemplate = `NAME:
{{.Name}} - {{.Usage}}

VERSION:
{{.Version}}

DESCRIPTION:
{{.Description}}

USAGE:
{{.Name}} {{if .Flags}}[flags] {{end}}command{{if .Flags}}{{end}} [arguments...]

COMMANDS:
	{{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
	{{end}}{{if .Flags}}
FLAGS:
	{{range .Flags}}{{.}}
	{{end}}{{end}}

`
)

func main() {
	app := cli.NewApp()
	app.Name = "Shogun"
	app.Author = "Ewetumo Alexander"
	app.Email = "trinoxf@gmail.com"
	app.Usage = "shogun {{command}}"
	app.Flags = []cli.Flag{}
	app.Description = "Shogun: Become one with your katana functions"
	app.CustomAppHelpTemplate = helpTemplate
	app.Action = mainAction

	app.Commands = []cli.Command{
		{
			Name:   "init",
			Action: initAction,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "n,name",
					Usage: "-name=bob-build",
				},
				cli.BoolFlag{
					Name:  "nopkg",
					Usage: "-nopkg=true",
				},
			},
		},
		{
			Name:   "list",
			Action: listAction,
			Flags:  []cli.Flag{},
		},
		{
			Name:   "add",
			Action: addAction,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "dn,dirName",
					Usage: "-dirName=bob-build",
				},
			},
		},
		{
			Name:   "build",
			Action: buildAction,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "up,usePkg",
					Usage: "-usePkg=true",
				},
				cli.BoolFlag{
					Name:  "skip,skipbuild",
					Usage: "-skip=true",
				},
			},
		},
		{
			Name:   "version",
			Action: versionAction,
			Flags:  []cli.Flag{},
		},
	}

	app.RunAndExitOnError()
}

func addAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return errors.New("You are required to supply name of file without extension .eg kodachi-task")
	}

	packageName := shogunateDirName
	toDirName := c.String("dirName")

	if toDirName != "" {
		packageName = toDirName
	}

	packageDir := filepath.Join(shogunateDirName, toDirName)

	if !hasDir(shogunateDirName) && toDirName == "" {
		packageName = "main"
		packageDir = ""
	}

	var directives []gen.WriteDirective

	for i := 0; i < c.NArg(); i++ {
		arg := c.Args().Get(i)
		if arg == "" {
			continue
		}

		fileName := fmt.Sprintf("%s.go", arg)
		directives = append(directives, gen.WriteDirective{
			Dir:      packageDir,
			FileName: fileName,
			Writer: gen.SourceTextWith(
				string(templates.Must("shogun-add.tml")),
				template.FuncMap{},
				struct {
					Package string
				}{
					Package: packageName,
				},
			),
		})

	}

	if err := ast.SimpleWriteDirectives("./", false, directives...); err != nil {
		return err
	}

	return nil
}

func initAction(c *cli.Context) error {
	currentDir, err := os.Getwd()
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", currentDir))
		return err
	}

	sourceDir, err := srcpath.RelativeToSrc(currentDir)
	if err != nil {
		events.Emit(metrics.Errorf("Must be run within go src path: %+q", err).With("dir", currentDir))
		return err
	}

	ctx := build.Default
	pkg, err := ctx.ImportDir("./", build.FindOnly)
	if err != nil {
		return err
	}

	packageTemplate := "shogun-in-pkg.tml"

	storeDir := shogunateDirName
	packageName := shogunateDirName
	if c.Bool("nopkg") {
		storeDir = ""
		packageName = "main"
	}

	pkgName := pkg.Name
	if pkgName == "" {
		pkgName = filepath.Base(sourceDir)
	}

	binaryName := c.String("name")
	if binaryName == "" {
		binaryName = fmt.Sprintf("%s_shogun", pkgName)
	}

	directives := []gen.WriteDirective{
		{
			Dir:      storeDir,
			FileName: "katana.go",
			Writer: gen.SourceTextWith(
				string(templates.Must(packageTemplate)),
				template.FuncMap{},
				struct {
					Package    string
					BinaryName string
				}{
					Package:    packageName,
					BinaryName: binaryName,
				},
			),
		},
	}

	if err := ast.SimpleWriteDirectives("./", false, directives...); err != nil {
		events.Emit(metrics.Errorf("Failed to write changes to disk: %+q", err).With("dir", currentDir))
		return err
	}

	ignoreFile := filepath.Join(currentDir, ".gitignore")
	if _, err := os.Stat(ignoreFile); err != nil {
		if igerr := addtoGitIgnore(ignoreFile); igerr != nil {
			events.Emit(metrics.Errorf("Failed to add changes to .gitignore: %+q", igerr).With("dir", currentDir))
			return igerr
		}
	}

	ignoreFileData, err := ioutil.ReadFile(ignoreFile)
	if err != nil {
		events.Emit(metrics.Errorf("Failed to read data from .gitignore: %+q", err).With("dir", currentDir).With("git_ignore", ignoreFile))
		return err
	}

	if !bytes.Contains(ignoreFileData, []byte(ignoreAddition)) {
		if igerr := addtoGitIgnore(ignoreFile); igerr != nil {
			events.Emit(metrics.Errorf("Failed to add changes to .gitignore: %+q", igerr).With("dir", currentDir))
			return igerr
		}
	}

	return nil
}

func mainAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return nil
	}

	var response, responseErr bytes.Buffer
	lsCmd := exec.New(exec.Command(""), exec.Async(), exec.Output(&response), exec.Err(&responseErr))

	_ = lsCmd
	return nil
}

func listAction(c *cli.Context) error {
	var targetDir string
	var hasShogunateDir bool

	currentDir, err := os.Getwd()
	if err != nil {
		events.Emit(metrics.Errorf("Failed to read current directory: %q", err))
		return err
	}

	targetDir = currentDir
	if shogunate := filepath.Join(currentDir, shogunateDirName); hasDir(shogunate) {
		hasShogunateDir = true
		targetDir = shogunate
	}

	ctx := build.Default
	ctx.BuildTags = append(ctx.BuildTags, "shogun")
	ctx.RequiredTags = append(ctx.RequiredTags, "shogun")

	// Build shogunate directory itself first.
	functions, err := buildFunctionList(targetDir, currentDir, ctx)
	if err != nil {
		events.Emit(metrics.Errorf("Failed to generate function list : %+q", err))
		return err
	}

	if hasShogunateDir {
		vdir := filepath.Join(currentDir, shogunateDirName)
		if err := vfiles.WalkDirSurface(vdir, func(rel string, abs string, info os.FileInfo) error {
			if !info.IsDir() {
				return nil
			}

			res, err := buildFunctionList(filepath.Join(vdir, rel), currentDir, ctx)
			if err != nil {
				return err
			}

			functions = append(functions, res...)
			return nil
		}); err != nil {
			events.Emit(metrics.Error(err).With("dir", currentDir).With("directory-visited", vdir))
			return err
		}
	}

	_ = functions
	return nil
}

func buildAction(c *cli.Context) error {
	skipBuild := c.Bool("skipbuild")

	var targetDir string
	var hasShogunateDir bool

	binaryPath := binPath()
	currentDir, err := os.Getwd()
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath))
		return err
	}

	targetDir = currentDir

	if shogunate := filepath.Join(currentDir, shogunateDirName); hasDir(shogunate) {
		hasShogunateDir = true
		targetDir = shogunate
	}

	ctx := build.Default
	ctx.BuildTags = append(ctx.BuildTags, "shogun")
	ctx.RequiredTags = append(ctx.RequiredTags, "shogun")

	// Build shogunate directory itself first.
	directives, err := buildDir(targetDir, currentDir, binaryPath, skipBuild, ctx)
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath).With("doSubDir", hasShogunateDir))
		return err
	}

	if hasShogunateDir {
		vdir := filepath.Join(currentDir, shogunateDirName)
		if err := vfiles.WalkDirSurface(vdir, func(rel string, abs string, info os.FileInfo) error {
			if !info.IsDir() {
				return nil
			}

			subdirqs, err := buildDir(filepath.Join(targetDir, rel), currentDir, binaryPath, skipBuild, ctx)
			if err != nil {
				events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath).With("doSubDir", hasShogunateDir))
				return err
			}

			directives = append(directives, subdirqs...)
			return nil
		}); err != nil {
			events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath).With("doSubDir", hasShogunateDir).With("directory-visited", vdir))
			return err
		}
	}

	if err := ast.SimpleWriteDirectives("./", true, directives...); err != nil {
		events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath).With("doSubDir", hasShogunateDir))
		return err
	}

	return nil
}

func buildDir(dir string, currentDir string, binaryPath string, skipBuild bool, ctx build.Context) ([]gen.WriteDirective, error) {
	pkgs, err := ast.FilteredPackageWithBuildCtx(nolog, dir, ctx)
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath))
		return nil, err
	}

	var directives []gen.WriteDirective

	for _, pkgItem := range pkgs {
		pkgHash, err := generateHash(pkgItem.Files)
		if err != nil {
			events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath))
			return nil, err
		}

		var binaryName, binaryExeName string
		if binAnnons := pkgItem.AnnotationsFor("@binaryName"); len(binAnnons) != 0 {
			if len(binAnnons[0].Arguments) == 0 {
				err := fmt.Errorf("binaryName annotation requires a single argument has the name of binary file")
				events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath).With("package", pkgItem.Package))
				continue
			}

			binaryName = strings.ToLower(binAnnons[0].Param("name"))
		} else {
			binaryName = pkgItem.Name
		}

		binaryExeName = binaryName
		if goosRuntime == "windows" {
			binaryExeName = fmt.Sprintf("%s.exec", binaryName)
		}

		// if the current hash we received is exactly like current calculated hash then continue to another package.
		if currentBinHash, err := binHash(filepath.Join(binaryPath, binaryName)); err == nil && currentBinHash == pkgHash {
			continue
		}

		packageBinaryPath := filepath.Join(".shogun", binaryName)

		for _, declr := range pkgItem.Packages {
			source := strings.Replace(declr.Source, strings.Join(declr.Comments, "\n"), "", -1)
			packageIndex := strings.Index(source, "package")
			packagePart := packageReg.FindString(source)

			source = source[packageIndex:]
			source = strings.TrimSpace(strings.Replace(source, packagePart, "", 1))

			directives = append(directives, gen.WriteDirective{
				FileName: filepath.Base(declr.FilePath),
				Dir:      packageBinaryPath,
				Writer: gen.SourceTextWith(
					string(templates.Must("shogun-src.tml")),
					template.FuncMap{},
					struct {
						Source string
					}{
						Source: source,
					},
				),
			})
		}

		directives = append(directives, gen.WriteDirective{
			FileName: "main.go",
			Dir:      packageBinaryPath,
			Writer: gen.SourceTextWith(
				string(templates.Must("shogun-main.tml")),
				template.FuncMap{},
				struct {
				}{},
			),
			After: func() error {
				if skipBuild {
					return nil
				}

				fmt.Printf("----------------------------------------\n")
				fmt.Printf("Building binary for shogunate: %q\n", binaryName)

				if err := exec.New(exec.Command("go build -x -o %s %s", filepath.Join(binaryPath, binaryExeName), filepath.Join(currentDir, packageBinaryPath, "main.go")), exec.Async()).Exec(context.Background(), nolog); err != nil {
					fmt.Printf("Building binary for shogun %q failed\n", binaryName)
					return err
				}

				fmt.Printf("Built binary for shogun %q into %q\n", binaryName, binaryPath)

				fmt.Printf("Cleaning up shogun binary build files... %q\n", binaryName)
				if err := os.RemoveAll(filepath.Join(currentDir, packageBinaryPath)); err != nil {
					fmt.Printf("Failed to properly cleanup build files %q\n", binaryName)
					return err
				}

				fmt.Printf("Shogun %q build ready\n\n", binaryName)
				return nil
			},
		})
	}

	return directives, nil
}

func buildFunctionList(dir string, currentDir string, ctx build.Context) ([]internal.Function, error) {
	pkgs, err := ast.FilteredPackageWithBuildCtx(nolog, dir, ctx)
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", currentDir))
		return nil, err
	}

	var functions []internal.Function

	for _, pkgItem := range pkgs {
		for _, declr := range pkgItem.Packages {
			for _, function := range declr.Functions {
				def, err := function.Definition()
				if err != nil {
					return nil, err
				}

				var fn internal.Function
				fn.Name = def.Name
				fn.Description = function.Comments
				fn.Synopses = doc.Synopsis(function.Comments)

				argLen := len(def.Args)
				retLen := len(def.Returns)

				switch argLen {
				case 0:
					switch retLen {
					case 0:
						fn.Type = internal.NoValue
					case 1:
						ret := def.Returns[0]
						if ret.Type != "error" {
							return nil, fmt.Errorf("Function %q from %q returns a type that is not an error", function.FuncName, function.FilePath)
						}

						fn.Type = internal.NoInErrReturn
					}
				case 1:
					arg := def.Args[0]

					switch arg.Type {
					case "internal.CancelContext":
						switch retLen {
						case 0:
							fn.Type = internal.CancelContextInNoErrReturn
						case 1:
							ret := def.Returns[0]
							if ret.Type != "error" {
								return nil, fmt.Errorf("Function %q from %q returns a type that is not an error", function.FuncName, function.FilePath)
							}

							fn.Type = internal.CancelContextInErrReturn
						}
					default:
						return nil, errors.New("Only internal.CancelContext are allowed")
					}
				case 2:
					arg := def.Args[0]

					switch arg.Type {
					case "internal.CancelContext":
						switch retLen {
						case 0:
							fn.Type = internal.CancelContextInNoErrReturn
						case 1:
							ret := def.Returns[0]
							if ret.Type != "error" {
								return nil, fmt.Errorf("Function %q from %q returns a type that is not an error", function.FuncName, function.FilePath)
							}

							fn.Type = internal.CancelContextInErrReturn
						}
					case "io.Reader":
						switch argLen {
						case 0:
							switch retLen {
							case 0:
								fn.Type = internal.ReaderInNoErrReturn
							case 1:
								ret := def.Returns[0]
								if ret.Type != "error" {
									return nil, fmt.Errorf("Function %q from %q returns a type that is not an error", function.FuncName, function.FilePath)
								}

								fn.Type = internal.ReaderInErrReturn
							}
						case 1:
						case 2:
							arg2 := def.Args[1]

							if arg2.Type != "io.WriterCloser" {
								return nil, fmt.Errorf("Function %q from %q must match\n - func(io.Reader, io.WriteCloser)\n - func(internal.CancelContext, io.Reader, io.WriteCloser)", function.FuncName, function.FilePath)
							}

						}
					case "map[string]interface{}":
					default:
					}
				case 3:
				}

				functions = append(functions, fn)
			}
		}
	}

	return functions, nil
}

func addtoGitIgnore(ignoreFile string) error {
	gitignore, err := os.OpenFile(ignoreFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer gitignore.Close()
	gitignore.Write([]byte(ignoreAddition))
	gitignore.Write([]byte("\n"))
	return nil
}

func hasDir(dir string) bool {
	if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
		return true
	}

	return false
}

func hasFile(file string) bool {
	if _, err := os.Stat(file); err == nil {
		return true
	}

	return false
}

func binHash(binPath string) (string, error) {
	var response bytes.Buffer

	if err := exec.New(exec.Command("%s hash", binPath), exec.Async(), exec.Output(&response)).Exec(context.Background(), nolog); err != nil {
		return "", err
	}

	return strings.TrimSpace(response.String()), nil
}

func versionAction(c *cli.Context) {
	fmt.Println(color.BlueString(fmt.Sprintf("shogun %s %s/%s", Version, runtime.GOOS, runtime.GOARCH)))
}

func binPath() string {
	shogunBinPath := os.Getenv("SHOGUNBIN")
	gobin := os.Getenv("GOBIN")
	gopath := os.Getenv("GOPATH")

	if runtime.GOOS == "windows" {
		gobin = filepath.ToSlash(gobin)
		gopath = filepath.ToSlash(gopath)
		shogunBinPath = filepath.ToSlash(shogunBinPath)
	}

	if shogunBinPath == "" && gobin == "" {
		return fmt.Sprintf("%s/bin", gopath)
	}

	if shogunBinPath == "" && gobin != "" {
		return gobin
	}

	return shogunBinPath
}

func generateHash(files []string) (string, error) {
	var hashes []string

	for _, file := range files {
		hash, err := generateFileHash(file)
		if err != nil {
			return "", err
		}

		hashes = append(hashes, hash)
	}

	return strings.Join(hashes, ""), nil
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
