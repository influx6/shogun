package main

//go:generate go generate ./templates/...

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
	"github.com/influx6/gobuild/build"
	"github.com/influx6/moz/ast"
	"github.com/influx6/moz/gen"
	"github.com/influx6/shogun/internal/samurai"
	"github.com/influx6/shogun/templates"
	"github.com/minio/cli"
)

// vars
var (
	Version          = "0.0.1"
	shogunateDirName = "katanas"
	ignoreAddition   = ".shogun"
	goosRuntime      = runtime.GOOS
	packageReg       = regexp.MustCompile(`package \w+`)
	binNameReg       = regexp.MustCompile("\\W+")
	helpTemplate     = `NAME:
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
	app.Version = Version
	app.Name = "Shogun"
	app.Author = "Ewetumo Alexander"
	app.Email = "trinoxf@gmail.com"
	app.Usage = "shogun {{command}}"
	app.Description = "Become one with your functions"
	app.CustomAppHelpTemplate = helpTemplate
	app.Action = mainAction

	app.Commands = []cli.Command{
		{
			Name:   "add",
			Action: addAction,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "dn,dirName",
					Usage: "-dirName=bob-build set the name of directory and package",
				},
				cli.BoolFlag{
					Name:  "v,verbose",
					Usage: "-verbose to show hidden logs and operations",
				},
			},
		},
		{
			Name:   "list",
			Action: listAction,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "d,dir",
					Usage: "-dir=./example to set directory to scan for functions",
				},
				cli.BoolFlag{
					Name:  "v,verbose",
					Usage: "-verbose to show hidden logs and operations",
				},
			},
		},
		{
			Name:   "build",
			Action: buildAction,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "d,dir",
					Value: "",
					Usage: "-dir=./katanas to build specific directory instead of root.",
				},
				cli.StringFlag{
					Name:  "cmd,cmdDir",
					Value: "",
					Usage: "-cmd=./cmd to build CLI package files into relative directory.",
				},
				cli.BoolFlag{
					Name:  "single,singlePkg",
					Usage: "-singlePkg=true to only bundle seperate command binaries",
				},
				cli.BoolFlag{
					Name:  "skip,skipbuild",
					Usage: "-skip to generate CLI package files without building binaries",
				},
				cli.BoolFlag{
					Name:  "v,verbose",
					Usage: "-verbose to show hidden logs and operations",
				},
				cli.BoolFlag{
					Name:  "f,force",
					Usage: "-f to force rebuild of binary",
				},
			},
		},
		{
			Name:   "help",
			Action: helpAction,
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
		return errors.New("You are required to supply spaced names of files .e.g signup.go signin.go run.go")
	}

	var packageName string

	packageDir := c.String("dirName")
	if packageDir == "" {
		packageName = "main"
	} else {
		packageName = toPackageName(packageDir)
	}

	var directives []gen.WriteDirective

	for i := 0; i < c.NArg(); i++ {
		arg := c.Args().Get(i)
		if arg == "" {
			continue
		}

		arg = strings.TrimSuffix(arg, filepath.Ext(arg))
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

func helpAction(c *cli.Context) error {
	if c.NArg() == 0 || c.Args().First() == "" {
		return nil
	}

	events := metrics.New()

	if c.Bool("verbose") {
		events = metrics.New(custom.StackDisplay(os.Stdout))
	}

	binaryPath := binPath()

	var response, responseErr bytes.Buffer
	binCmd := exec.New(
		exec.Async(),
		exec.Command("%s/%s help %s", binaryPath, c.Args().First(), strings.Join(c.Args().Tail(), " ")),
		exec.Output(&response),
		exec.Err(&responseErr),
		exec.Input(os.Stdout),
	)

	if err := binCmd.Exec(context.Background(), events); err != nil {
		events.Emit(metrics.Error(err))
		return fmt.Errorf("Command Error: %+q", err)
	}

	return nil
}

func mainAction(c *cli.Context) error {
	if c.NArg() == 0 || c.Args().First() == "" {
		fmt.Println("Nothing to do...")
		return nil
	}

	events := metrics.New()

	if c.Bool("verbose") {
		events = metrics.New(custom.StackDisplay(os.Stdout))
	}

	binaryPath := binPath()

	var response, responseErr bytes.Buffer
	binCmd := exec.New(
		exec.Async(),
		exec.Command("%s/%s %s", binaryPath, c.Args().First(), strings.Join(c.Args().Tail(), " ")),
		exec.Output(&response),
		exec.Err(&responseErr),
		exec.Input(os.Stdout),
	)

	if err := binCmd.Exec(context.Background(), events); err != nil {
		events.Emit(metrics.Error(err))
		return fmt.Errorf("Command Error: %+q\n %+s", err, responseErr.String())
	}

	fmt.Println(response.String())
	return nil
}

func listAction(c *cli.Context) error {
	events := metrics.New()

	if c.Bool("verbose") {
		events = metrics.New(custom.StackDisplay(os.Stdout))
	}

	currentDir, err := os.Getwd()
	if err != nil {
		events.Emit(metrics.Errorf("Failed to read current directory: %q", err))
		return err
	}

	ctx := build.Default
	ctx.BuildTags = append(ctx.BuildTags, "shogun")
	ctx.RequiredTags = append(ctx.RequiredTags, "shogun")

	// Build shogunate directory itself first.
	functions, err := samurai.ListFunctions(events, events, filepath.Join(currentDir, c.String("dir")), ctx)
	if err != nil {
		events.Emit(metrics.Errorf("Failed to generate function list : %+q", err))
		return fmt.Errorf("Not a shogun directory or contains no shogun files: %+q", err)
	}

	result := gen.SourceTextWithName(
		"shogun-pkg-list",
		string(templates.Must("shogun-pkg-list.tml")),
		template.FuncMap{},
		functions,
	)

	_, err = result.WriteTo(os.Stdout)
	if err != nil {
		return err
	}

	return nil
}

func buildAction(c *cli.Context) error {
	events := metrics.New()

	if c.Bool("verbose") {
		events = metrics.New(custom.StackDisplay(os.Stdout))
	}

	skipBuild := c.Bool("skipbuild")
	forceBuild := c.Bool("force")
	tgDir := c.String("dir")
	cmdDir := c.String("cmdDir")

	binaryPath := binPath()
	currentDir, err := os.Getwd()
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath))
		return err
	}

	if igerr := checkAndAddIgnore(currentDir); igerr != nil {
		events.Emit(metrics.Errorf("Failed to add changes to .gitignore: %+q", igerr).With("dir", currentDir))
		return igerr
	}

	targetDir := filepath.Join(currentDir, tgDir)

	if cmdDir == "" {
		cmdDir = filepath.Join(".shogun", "cmd")
	}

	ctx := build.Default
	ctx.BuildTags = append(ctx.BuildTags, "shogun")
	ctx.RequiredTags = append(ctx.RequiredTags, "shogun")

	// Build hash list for directories.
	hashList, err := samurai.ListPackageHash(events, events, targetDir, ctx)
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath))
		return err
	}

	// Build directories for commands.
	directive, err := samurai.BuildPackage(events, events, targetDir, cmdDir, currentDir, binaryPath, skipBuild, c.Bool("singlePkg"), ctx)
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath))
		return err
	}

	var subUpdated bool

	for _, sub := range directive.Subs {
		if hashData, ok := hashList.Subs[sub.Path]; ok {
			hashFile := filepath.Join(currentDir, sub.PkgPath, ".hashfile")
			prevHash, err := readFile(hashFile)

			if err == nil && prevHash == hashData.Hash && !forceBuild {
				continue
			}
		}

		// if PkgPath is empty then possibly not one we want to handle, all must
		// have a place to store
		if sub.PkgPath == "" || len(sub.List) == 0 {
			continue
		}

		subUpdated = true
		if err := ast.SimpleWriteDirectives("./", true, sub.List...); err != nil {
			events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath))
			return err
		}
	}

	// Validate hash of main cmd.
	if !forceBuild {
		hashFile := filepath.Join(currentDir, directive.Main.PkgPath, ".hashfile")
		if prevHash, err := readFile(hashFile); err == nil && prevHash == hashList.Main.Hash && !subUpdated {
			return nil
		}
	}

	if err := ast.SimpleWriteDirectives("./", true, directive.Main.List...); err != nil {
		events.Emit(metrics.Error(err).With("dir", currentDir).With("binary_path", binaryPath))
		return err
	}

	return nil
}

func checkAndAddIgnore(currentDir string) error {
	ignoreFile := filepath.Join(currentDir, ".gitignore")
	if _, ierr := os.Stat(ignoreFile); ierr != nil {
		if igerr := addtoGitIgnore(ignoreFile); igerr != nil {
			return igerr
		}
	}

	ignoreFileData, err := ioutil.ReadFile(ignoreFile)
	if err != nil {
		return err
	}

	if !bytes.Contains(ignoreFileData, []byte(ignoreAddition)) {
		if igerr := addtoGitIgnore(ignoreFile); igerr != nil {
			return igerr
		}
	}

	return nil
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

func readFile(file string) (string, error) {
	content, err := ioutil.ReadFile(file)
	return string(bytes.TrimSpace(content)), err
}

func hasFile(file string) bool {
	if _, err := os.Stat(file); err == nil {
		return true
	}

	return false
}

func versionAction(c *cli.Context) {
	fmt.Println(color.BlueString(fmt.Sprintf("shogun %s %s/%s", Version, runtime.GOOS, runtime.GOARCH)))
}

func toPackageName(name string) string {
	return strings.ToLower(binNameReg.ReplaceAllString(name, ""))
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
