package main

//go:generate go generate ./templates/...

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
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
	"github.com/influx6/gobuild/build"
	"github.com/influx6/gobuild/srcpath"
	"github.com/influx6/moz/ast"
	"github.com/influx6/moz/gen"
	"github.com/influx6/shogun/templates"
	"github.com/minio/cli"
)

// Version defines the version number for the cli.
var Version = "0.1"
var nolog = metrics.New()
var shogunateDirName = "katanas"
var goosRuntime = runtime.GOOS
var events = metrics.New(custom.StackDisplay(os.Stdout))
var packageReg = regexp.MustCompile(`package \w+`)

var helpTemplate = `NAME:
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
			Name:   "add",
			Action: addAction,
			Flags:  []cli.Flag{},
		},
		{
			Name:   "build",
			Action: buildAction,
			Flags:  []cli.Flag{},
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
	packageDir := shogunateDirName
	fileName := fmt.Sprintf("%s.go", c.Args().First())

	if !hasDir(shogunateDirName) {
		packageName = "main"
		packageDir = ""
	}

	directives := []gen.WriteDirective{
		{
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
		},
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
		return err
	}

	gitignore, err := os.OpenFile(filepath.Join(currentDir, ".gitignore"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer gitignore.Close()
	gitignore.Write([]byte(".shogun\n"))

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

func buildAction(c *cli.Context) error {
	var targetDir string

	binaryPath := binPath()
	currentDir, err := os.Getwd()
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath))
		return err
	}

	targetDir = currentDir
	if shogunate := filepath.Join(currentDir, shogunateDirName); hasDir(shogunate) {
		targetDir = shogunate
	}

	ctx := build.Default
	ctx.BuildTags = append(ctx.BuildTags, "shogun")
	ctx.RequiredTags = append(ctx.RequiredTags, "shogun")
	pkgs, err := ast.FilteredPackageWithBuildCtx(nolog, targetDir, ctx)
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath))
		return err
	}

	var directives []gen.WriteDirective

	for _, pkgItem := range pkgs {
		pkgHash, err := generateHash(pkgItem.Files)
		if err != nil {
			events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath))
			return err
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
				string(templates.Must("main.tml")),
				template.FuncMap{},
				struct {
				}{},
			),
			After: func() error {
				fmt.Printf("Building binary for shogun %q\n", binaryName)

				if err := exec.New(exec.Command("go build -x -o %s %s", filepath.Join(binaryPath, binaryExeName), filepath.Join(currentDir, packageBinaryPath, "main.go")), exec.Async()).Exec(context.Background(), nolog); err != nil {
					fmt.Printf("Building binary for shogun %q failed\n", binaryName)
					return err
				}

				fmt.Printf("Built binary for shogun %q into %q\n", binaryName, binaryPath)

				fmt.Printf("Cleaning up shogun binary build files... %q\n", binaryName)
				if err := os.Remove(filepath.Join(currentDir, packageBinaryPath)); err != nil {
					fmt.Printf("Failed to proper clean up shogun binary build files %q\n", binaryName)
					return err
				}

				fmt.Printf("Shogun %q build ready\n\n", binaryName)
				return nil
			},
		})
	}

	if err := ast.SimpleWriteDirectives("./", true, directives...); err != nil {
		return err
	}

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
