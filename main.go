package main

//go:generate go generate ./...

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/influx6/faux/exec"
	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/metrics/custom"
	"github.com/influx6/gobuild/build"
	"github.com/influx6/moz/ast"
	"github.com/influx6/moz/gen"
	"github.com/influx6/shogun/templates"
	"github.com/minio/cli"
)

// Version defines the version number for the cli.
var Version = "0.1"
var shogunateDirName = "shogunate"
var events = metrics.New(custom.StackDisplay(os.Stdout))

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
				cli.BoolFlag{
					Name:  "nosh,noshogunate",
					Usage: "-nosh=true",
				},
			},
		},
		{
			Name:   "build",
			Action: buildit,
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

func initAction(c *cli.Context) error {
	ctx := build.Default
	pkg, err := ctx.ImportDir("./", build.FindOnly)
	if err != nil {
		return err
	}

	var storeDir = ""
	var packageName = pkg.Name

	if c.Bool("noshogunate") {
		storeDir = shogunateDirName
		packageName = "shogunate"
	}

	directives := []gen.WriteDirective{
		{
			Dir:      storeDir,
			FileName: "shogun.go",
			Writer: gen.SourceTextWith(
				string(templates.Must("shogunate-main.tml")),
				template.FuncMap{},
				struct {
					Package string
				}{
					Package: packageName,
				},
			),
		},
	}

	return ast.SimpleWriteDirectives("./", false, directives...)
}

func mainAction(c *cli.Context) error {
	var response, responseErr bytes.Buffer
	lsCmd := exec.New(exec.Command(""), exec.Async(), exec.Output(&response), exec.Err(&responseErr))

	_ = lsCmd
	return nil
}

func buildit(c *cli.Context) error {
	var targetDir string

	binaryPath := binPath()
	currentDir, err := os.Getwd()
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath))
		return err
	}

	targetDir = currentDir

	if stat, err := os.Stat(filepath.Join(currentDir, shogunateDirName)); err == nil && stat.IsDir() {
		targetDir = filepath.Join(currentDir, shogunateDirName)
	}

	ctx := build.Default
	ctx.BuildTags = append(ctx.BuildTags, "shogun")
	ctx.RequiredTags = append(ctx.RequiredTags, "shogun")
	pkg, err := ast.FilteredPackageWithBuildCtx(events, targetDir, ctx)
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath))
		return err
	}

	for _, pkgItem := range pkg.Packages {
		pkgHash, err := generateHash(pkgItem.Files)
		if err != nil {
			events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath))
			return err
		}

		var binaryName string

		if binAnnons := pkgItem.AnnotationsFor("@binaryName"); len(binAnnons) != 0 {
			if len(binAnnons[0].Arguments) == 0 {
				err := fmt.Errorf("binaryName annotation requires a single argument has the name of binary file")
				events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath).With("package", pkgItem.Package))
				continue
			}

			binaryName = binAnnons[0].Arguments[0]
		} else {
			binaryName = strings.ToLower(filepath.Base(pkgItem.Package))
		}

		// if the current hash we received is exactly like current calculated hash then continue to another package.
		if currentBinHash, err := binHash(filepath.Join(binaryPath, binaryName)); err == nil && currentBinHash == pkgHash {
			continue
		}
	}

	return nil
}

func binHash(binPath string) (string, error) {
	var response bytes.Buffer

	if err := exec.New(exec.Command("%s hash", binPath), exec.Async(), exec.Output(&response)).Exec(context.Background(), events); err != nil {
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
