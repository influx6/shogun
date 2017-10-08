package samurai

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"

	"github.com/influx6/faux/exec"
	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/vfiles"
	"github.com/influx6/gobuild/build"
	"github.com/influx6/moz/ast"
	"github.com/influx6/moz/gen"
	"github.com/influx6/shogun/templates"
)

var (
	ignoreAddition = ".shogun"
	goosRuntime    = runtime.GOOS
	packageReg     = regexp.MustCompile(`package \w+`)
)

// BuildPackage builds a shogun binarie commandline files for giving directory and 1 level directory.
func BuildPackage(vlog metrics.Metrics, events metrics.Metrics, dir string, binaryPath string, skipBuild bool, ctx build.Context) ([]gen.WriteDirective, error) {
	directives, err := BuildPackageForDir(vlog, events, dir, binaryPath, skipBuild, ctx)
	if err != nil {
		events.Emit(metrics.Error(err).With("dir", dir).With("binary_path", binaryPath))
		return nil, err
	}

	if err := vfiles.WalkDirSurface(dir, func(rel string, abs string, info os.FileInfo) error {
		if !info.IsDir() {
			return nil
		}

		res, err := BuildPackageForDir(vlog, events, abs, binaryPath, skipBuild, ctx)
		if err != nil {
			return err
		}

		directives = append(directives, res...)
		return nil
	}); err != nil {
		events.Emit(metrics.Error(err).With("dir", dir))
		return directives, err
	}

	return directives, nil
}

// BuildPackageForDir generates needed package files for creating new function based executable binaries.
func BuildPackageForDir(vlog metrics.Metrics, events metrics.Metrics, dir string, binaryPath string, skipBuild bool, ctx build.Context) ([]gen.WriteDirective, error) {
	pkgs, err := ast.FilteredPackageWithBuildCtx(vlog, dir, ctx)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			return nil, nil
		}

		events.Emit(metrics.Error(err).With("dir", dir).With("binary_path", binaryPath))
		return nil, err
	}

	var directives []gen.WriteDirective

	for _, pkgItem := range pkgs {
		pkgHash, err := generateHash(pkgItem.Files)
		if err != nil {
			events.Emit(metrics.Error(err).With("dir", dir).With("binary_path", binaryPath))
			return nil, err
		}

		var binaryName, binaryExeName string
		if binAnnons := pkgItem.AnnotationsFor("@binaryName"); len(binAnnons) != 0 {
			if len(binAnnons[0].Arguments) == 0 {
				err := fmt.Errorf("binaryName annotation requires a single argument has the name of binary file")
				events.Emit(metrics.Error(err).With("dir", dir).With("binary_path", binaryPath).With("package", pkgItem.Path))
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
		if currentBinHash, err := binHash(vlog, filepath.Join(binaryPath, binaryName)); err == nil && currentBinHash == pkgHash {
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

				if err := exec.New(exec.Command("go build -x -o %s %s", filepath.Join(binaryPath, binaryExeName), filepath.Join(dir, packageBinaryPath, "main.go")), exec.Async()).Exec(context.Background(), vlog); err != nil {
					fmt.Printf("Building binary for shogun %q failed\n", binaryName)
					return err
				}

				fmt.Printf("Built binary for shogun %q into %q\n", binaryName, binaryPath)

				fmt.Printf("Cleaning up shogun binary build files... %q\n", binaryName)
				if err := os.RemoveAll(filepath.Join(dir, packageBinaryPath)); err != nil {
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

func binHash(nlog metrics.Metrics, binPath string) (string, error) {
	var response bytes.Buffer

	if err := exec.New(exec.Command("%s hash", binPath), exec.Async(), exec.Output(&response)).Exec(context.Background(), nlog); err != nil {
		return "", err
	}

	return strings.TrimSpace(response.String()), nil
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
